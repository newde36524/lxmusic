package lx

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

/*
User sources — the "自定义音源" feature of the desktop app's settings page.

Nothing in dist/renderer.js contains an audio URL endpoint for any platform.
The renderer only builds a request like

    { source: "kw", action: "musicUrl", info: {...} }

and, when the selected source id starts with "user_api", it is answered inside a
hidden BrowserWindow with Node enabled, which evaluates the imported script. The
same is true for action "lyric" and "pic": the endpoints all live in the
third-party scripts.

This file reproduces that contract by running the script in a `node` subprocess
instead of an Electron window. The script is cached locally, a small bridge
module feeds it one JSON request on stdin and reads one JSON answer from stdout.
Sources work unmodified — obfuscated ones included — because the VM is the real
one; only the host changes from Electron to a plain node process.
*/

// userBridge is the module that runs inside node. No backticks: it is a Go raw
// string literal. It shims the two npm packages community sources reach for,
// so the script needs no `npm install`.
const userBridge = `
var fs = require("fs");
var Module = require("module");
var httpLib = require("http");
var httpsLib = require("https");
var crypto = require("crypto");
var path = require("path");

// lastFailure remembers the most recent non-2xx reply so a source that fails
// without a message can still be diagnosed from the outside.
var lastFailure = null;

function doRequest(method, target, opts) {
  opts = opts || {};
  return new Promise(function (resolve, reject) {
    var u = new URL(target);
    var lib = u.protocol === "https:" ? httpsLib : httpLib;
    var body = opts.body;
    var req = lib.request({
      method: method || "GET",
      hostname: u.hostname,
      port: u.port,
      path: u.pathname + u.search,
      headers: opts.headers || {},
      rejectUnauthorized: false,
    }, function (res) {
      var chunks = [];
      res.on("data", function (c) { chunks.push(c); });
      res.on("end", function () {
        var buf = Buffer.concat(chunks);
        var out = { statusCode: res.statusCode, status: res.statusCode, headers: res.headers, body: buf, rawBody: buf };
        if (opts.json) { try { out.body = JSON.parse(buf.toString("utf8")); } catch (e) {} }
        if (res.statusCode < 200 || res.statusCode >= 300) {
          lastFailure = { method: method || "GET", url: target, status: res.statusCode };
        }
        resolve(out);
      });
    });
    req.setTimeout(opts.timeout || 20000, function () { req.destroy(new Error("request timeout")); });
    req.on("error", reject);
    if (body != null && body !== "") req.write(body);
    req.end();
  });
}

// The "request" npm package, which most lx-music sources require.
function requestFn(target, opts) {
  if (typeof target !== "string") { opts = target || {}; target = opts.url; }
  opts = opts || {};
  if (opts.form && !opts.body) {
    var parts = [];
    for (var k in opts.form) parts.push(encodeURIComponent(k) + "=" + encodeURIComponent(opts.form[k]));
    opts.body = parts.join("&");
    opts.headers = Object.assign({}, opts.headers, { "Content-Type": "application/x-www-form-urlencoded" });
  }
  if (opts.qs) {
    var q = [];
    for (var j in opts.qs) q.push(encodeURIComponent(j) + "=" + encodeURIComponent(opts.qs[j]));
    if (q.length) target += (target.indexOf("?") < 0 ? "?" : "&") + q.join("&");
  }
  return doRequest(opts.method || "GET", target, opts).then(function (r) {
    var buf = Buffer.isBuffer(r.body) ? r.body : Buffer.from(String(r.body));
    var text = buf.toString("utf8");
    var out = { statusCode: r.statusCode, status: r.statusCode, headers: r.headers, body: text, raw: buf, rawBody: buf };
    try { out.json = JSON.parse(text); } catch (e) {}
    return out;
  });
}

function md5(s) { return crypto.createHash("md5").update(String(s)).digest("hex"); }
function cryptoJS() {
  return {
    MD5: function (s) { return { toString: function () { return md5(s); } }; },
    HmacMD5: function (k, s) { return { toString: function () { return crypto.createHash("md5").update(String(s)).update(String(k)).digest("hex"); } }; },
  };
}

// request.get / request.post / ... are the shorthand form most sources use.
function withMethod(method) {
  return function (target, opts) {
    return requestFn(target, Object.assign({ method: method }, opts || {}));
  };
}
requestFn.get = withMethod("GET");
requestFn.post = withMethod("POST");
requestFn.put = withMethod("PUT");
requestFn.del = requestFn.delete = withMethod("DELETE");
requestFn.defaults = function (o) { return requestFn; };

// axios resolves to { data, status, headers }; axios.create() gives the same.
function axiosFn(cfg) {
  cfg = (typeof cfg === "string") ? { url: cfg } : (cfg || {});
  return requestFn(cfg.url, cfg).then(function (r) {
    var b = r.body;
    try { b = JSON.parse(b); } catch (e) {}
    return { data: b, status: r.statusCode, statusText: "", headers: r.headers || {}, config: cfg };
  });
}
axiosFn.create = function () { return axiosFn; };
axiosFn.get = function (u, cfg) { return axiosFn(Object.assign({ url: u, method: "GET" }, cfg || {})); };
axiosFn.post = function (u, data, cfg) { return axiosFn(Object.assign({ url: u, method: "POST", body: data }, cfg || {})); };

// globalThis.lx is built by user-api-preload.js before the script runs. It is
// small: EVENT_NAMES, request(), send(), on(), utils and the script's own
// header. A published source destructures it at load time, so a missing member
// aborts the whole script.
//
// request(target, { method, timeout, headers, body, form, formData }, cb)
// answers through the callback with { statusCode, statusMessage, headers,
// bytes, raw, body } where body is JSON-parsed when it parses; the return
// value is a canceller.
function lxRequest(target, options, cb) {
  options = options || {};
  if (typeof options === "function") { cb = options; options = {}; }
  var method = String(options.method || "get").toUpperCase();
  var payload = options.body != null ? options.body : (options.form != null ? options.form : options.formData);
  var opts = { headers: Object.assign({}, options.headers || {}) };
  opts.timeout = typeof options.timeout === "number" && options.timeout > 0 ? Math.min(options.timeout, 60000) : 60000;

  if (Array.isArray(payload)) {
    // request's multipart form: [{ name, value, filename, type }]. Built as a
    // Buffer because a file part is not text.
    var boundary = "----lxmusic" + Date.now().toString(16);
    var parts = [];
    for (var i = 0; i < payload.length; i++) {
      var f = payload[i];
      parts.push(Buffer.from("--" + boundary + "\r\n"));
      var disp = "Content-Disposition: form-data; name=\"" + f.name + "\"";
      if (f.filename) disp += "; filename=\"" + f.filename + "\"";
      parts.push(Buffer.from(disp + "\r\n"));
      parts.push(Buffer.from("Content-Type: " + (f.type || "application/octet-stream") + "\r\n\r\n"));
      parts.push(Buffer.isBuffer(f.value) ? f.value : Buffer.from(String(f.value)));
      parts.push(Buffer.from("\r\n"));
    }
    parts.push(Buffer.from("--" + boundary + "--\r\n\r\n"));
    opts.body = Buffer.concat(parts);
    opts.headers["Content-Type"] = "multipart/form-data; boundary=" + boundary;
  } else if (payload != null && typeof payload === "object" && !Buffer.isBuffer(payload)) {
    // Plain objects are urlencoded unless a key says otherwise.
    var keys = Object.keys(payload);
    var simple = keys.length > 0 && keys.every(function (k) { return /^[a-zA-Z_][\w]*$/.test(k); });
    if (simple) {
      var enc = [];
      for (var j = 0; j < keys.length; j++) enc.push(encodeURIComponent(keys[j]) + "=" + encodeURIComponent(payload[keys[j]]));
      opts.body = enc.join("&");
      opts.headers["Content-Type"] = "application/x-www-form-urlencoded";
    } else {
      opts.body = JSON.stringify(payload);
      opts.headers["Content-Type"] = "application/json";
    }
  } else if (payload != null) {
    opts.body = payload;
  }

  var settled = false;
  var answer = function (err, res, data) {
    if (settled || !cb) return;
    settled = true;
    cb.call(null, err, res, data);
  };
  doRequest(method, target, opts).then(function (r) {
    var buf = Buffer.isBuffer(r.body) ? r.body : Buffer.from(r.body);
    var text = buf.toString("utf8");
    var body = text;
    try { body = JSON.parse(text); } catch (e) {}
    var res = {
      statusCode: r.statusCode,
      statusMessage: "",
      status: r.statusCode,
      headers: r.headers || {},
      bytes: text.length,
      raw: buf,
      body: body,
      getHeader: function (n) { return (r.headers || {})[String(n).toLowerCase()]; },
    };
    answer(null, res, body);
    return res;
  }, function (e) {
    answer(e, null, null);
    return Promise.reject(e);
  });
  return function () {};
}

// EVENT_NAMES is the tiny object the preload actually exposes. Sources compare
// against these strings when they call send() or on().
var EVENT_NAMES = { request: "request", inited: "inited", updateAlert: "updateAlert" };
var lxInited = false;
var lxAlerted = false;
function lxSend(name, payload) {
  if (name !== EVENT_NAMES.request && name !== EVENT_NAMES.inited && name !== EVENT_NAMES.updateAlert) {
    return Promise.reject(new Error("The event is not supported: " + name));
  }
  if (name === EVENT_NAMES.inited) {
    if (lxInited) return Promise.reject(new Error("Script is inited"));
    lxInited = true;
    return Promise.resolve();
  }
  if (name === EVENT_NAMES.updateAlert) {
    if (lxAlerted) return Promise.reject(new Error("The update alert can only be called once."));
    lxAlerted = true;
    if (!payload || typeof payload !== "object" || typeof payload.log !== "string") {
      return Promise.reject(new Error("parameter format error."));
    }
    return Promise.resolve();
  }
  return Promise.resolve();
}
// on() only ever accepts EVENT_NAMES.request; nothing is wired up here because
// there is no renderer to send a second request from.
function lxOn(name) {
  if (name !== EVENT_NAMES.request) return Promise.reject(new Error("The event is not supported: " + name));
  return Promise.resolve();
}

// utils is the crypto/buffer/zlib helper bundle the preload gives sources so
// they do not have to require the modules themselves.
var zlibLib = require("zlib");
var lxUtils = {
  crypto: {
    aesEncrypt: function (data, key, iv, mode) {
      var c = crypto.createCipheriv(mode, key, iv);
      return Buffer.concat([c.update(data), c.final()]);
    },
    aesDecrypt: function (data, key, iv, mode) {
      var c = crypto.createDecipheriv(mode, key, iv);
      return Buffer.concat([c.update(data), c.final()]);
    },
    rsaEncrypt: function (data, key) {
      data = Buffer.concat([Buffer.alloc(128 - data.length), data]);
      return crypto.publicEncrypt({ key: key, padding: crypto.constants.RSA_NO_PADDING }, data);
    },
    randomBytes: function (n) { return crypto.randomBytes(n); },
    md5: function (s) { return crypto.createHash("md5").update(s).digest("hex"); },
    sha1: function (s) { return crypto.createHash("sha1").update(s).digest("hex"); },
    sha256: function (s) { return crypto.createHash("sha256").update(s).digest("hex"); },
  },
  buffer: {
    from: function () { return Buffer.from.apply(Buffer, arguments); },
    bufToString: function (b, t) { return Buffer.from(b, "binary").toString(t); },
  },
  zlib: {
    inflate: function (b) {
      return new Promise(function (ok, no) { zlibLib.inflate(b, function (e, o) { e ? no(new Error(e.message)) : ok(o); }); });
    },
    deflate: function (b) {
      return new Promise(function (ok, no) { zlibLib.deflate(b, function (e, o) { e ? no(new Error(e.message)) : ok(o); }); });
    },
  },
};

// parseScriptInfo reads the /* @name ... */ header, the same way the app does.
function scriptInfo(script) {
  var info = { name: "", description: "", version: "", author: "", homepage: "", rawScript: script };
  var m = /^\s*\/\*[\s\S]+?\*\//.exec(script);
  if (!m) return info;
  var fields = [
    ["@description", "description"],
    ["@homepage", "homepage"],
    ["@name", "name"],
    ["@version", "version"],
    ["@author", "author"],
  ];
  var lines = m[0].split("\n");
  for (var i = 0; i < lines.length; i++) {
    var line = lines[i].replace(/^\s*\*[\s]?/, "").replace(/\s+$/, "");
    for (var j = 0; j < fields.length; j++) {
      if (line.indexOf(fields[j][0]) === 0) {
        var value = line.substring(fields[j][0].length).replace(/^[:\s-]+/, "").replace(/\s+$/, "");
        info[fields[j][1]] = value;
        break;
      }
    }
  }
  return info;
}

// currentScriptInfo is filled in once the script text is known, which happens
// after this preamble runs.
var scriptInfoHolder = { value: null };
var CURRENT_SCRIPT_INFO = new Proxy({ rawScript: "" }, {
  get: function (t, k) {
    if (scriptInfoHolder.value && k in scriptInfoHolder.value) return scriptInfoHolder.value[k];
    return t[k];
  },
});
var lx = {
  EVENT_NAMES: EVENT_NAMES,
  request: lxRequest,
  send: lxSend,
  on: lxOn,
  utils: lxUtils,
  currentScriptInfo: CURRENT_SCRIPT_INFO,
  version: "2.0.0",
  env: "desktop",
};
globalThis.lx = lx;

var SHIMS = {
  "request": requestFn,
  "axios": axiosFn,
  "md5": md5,
  "crypto-js": cryptoJS(),
};
var origLoad = Module._load;
Module._load = function (id, parent, isMain) {
  if (Object.prototype.hasOwnProperty.call(SHIMS, id)) return SHIMS[id];
  return origLoad.apply(this, arguments);
};

// stdout carries the protocol reply and nothing else. A source that console.logs
// its arguments (huibq does so for every musicUrl, via console.group) would
// otherwise append its debug text to the JSON reply and the parse would fail.
// The desktop app never sees this: a WebView's console output goes to DevTools,
// not to the protocol channel. Everything below is diverted to stderr instead.
// Each reply ends in a newline and JSON.stringify never emits a raw newline in a
// string, so one process can serve a stream of requests and Go can read them with
// ReadBytes.
function reply(v) { process.stdout.write(JSON.stringify(v) + "\n"); }
function dumpErr(a) {
  var s = typeof a === "string" ? a : JSON.stringify(a, null, 2);
  process.stderr.write((s || "") + "\n");
}
console.log = console.info = console.debug = console.warn = console.table = dumpErr;
console.group = function (m) { dumpErr(typeof m === "string" ? m : ""); };
console.groupEnd = function () { dumpErr(""); };

function loadScript(p) {
  var raw = fs.readFileSync(p, "utf8");
  scriptInfoHolder.value = scriptInfo(raw);
  // CommonJS first: lx-music sources usually end in module.exports = {...}.
  if (!/^\s*export\s/m.test(raw)) {
    var m = { exports: {} };
    var fn = new Function("module", "exports", "require", "__filename", "__dirname", raw);
    fn(m, m.exports, require, p, path.dirname(p));
    return m.exports;
  }
  // ESM sources: import them through a data: URL and inject a real require.
  var header = "import { createRequire } from 'module';\n" +
    "import { pathToFileURL } from 'url';\n" +
    "const require = createRequire(pathToFileURL(process.argv[2]).href);\n";
  var b64 = Buffer.from(header + raw, "utf8").toString("base64");
  return import("data:text/javascript;base64," + b64);
}

// The script announces itself with send(inited, payload) and answers requests
// through the handler passed to on(request, fn). Nothing is exported: the
// preload reads the init payload for the platform/quality table and then calls
// the handler with { source, action, info }.
var initPayload = null;
var requestHandler = null;
var initedResolve = null;
var inited = new Promise(function (resolve) { initedResolve = resolve; });
var initedDone = false;

lx.send = function (name, payload) {
  if (name === EVENT_NAMES.inited) {
    if (initPayload) return Promise.reject(new Error("Script is inited"));
    initPayload = payload;
    initedDone = true;
    if (initedResolve) initedResolve(true);
    return Promise.resolve();
  }
  if (name === EVENT_NAMES.updateAlert) return Promise.resolve();
  return Promise.reject(new Error("The event is not supported: " + name));
};
lx.on = function (name, fn) {
  if (name === EVENT_NAMES.request) { requestHandler = fn; return Promise.resolve(); }
  return Promise.reject(new Error("The event is not supported: " + name));
};

// errText turns a rejection into something a human can act on. A source that
// does Error(undefined) or Error("") collapses to the literal "Error", which
// is the least useful thing to print.
function errText(e) {
  if (e == null) return "the source failed without a message";
  if (typeof e === "string") return e;
  var msg = String(e.message || "");
  if (msg === "" || msg === "undefined" || msg === "null") return "the source failed without a message";
  return msg;
}
// failNote attaches the last failing upstream reply. A source usually rejects
// with an empty message right after its backend answers 4xx/5xx.
function failNote() {
  if (!lastFailure) return "";
  return " (last upstream reply: " + lastFailure.method + " " + lastFailure.url + " -> " + lastFailure.status + ")";
}

// QUALITY_ORDER is the app's quality ranking, best first.
var QUALITY_ORDER = ["flac24bit", "flac", "320k", "128k"];

// clampQuality settles on a quality the source actually declared. A source
// offering only 128k answers an "auto" or "flac" request with 404, so the
// request is rewritten to the highest quality it serves — the same "best
// available" rule the CLI applies to platform sources.
function clampQuality(action, info) {
  if (action !== "musicUrl" || !initPayload || !initPayload.sources || !info) return info;
  var cap = initPayload.sources[info.source];
  if (!cap || !cap.qualitys || !cap.qualitys.length) return info;
  var want = info.type;
  if (want && cap.qualitys.indexOf(want) >= 0) return info;
  var pick = cap.qualitys[0];
  for (var i = 0; i < QUALITY_ORDER.length; i++) {
    if (cap.qualitys.indexOf(QUALITY_ORDER[i]) >= 0) { pick = QUALITY_ORDER[i]; break; }
  }
  var out = Object.assign({}, info, { type: pick });
  if (out.musicInfo) out.musicInfo = Object.assign({}, out.musicInfo, { type: pick });
  return out;
}

// validateResult applies the preload's own guards, so a script that answers
// with garbage fails the same way it would inside the desktop app.
function validateResult(action, result) {
  if (action === "musicUrl" || action === "pic") {
    if (typeof result !== "string" || result.length > 2048 || !/^https?:/.test(result)) throw new Error("failed");
    return result;
  }
  if (action === "lyric") {
    if (typeof result !== "object" || result === null || typeof result.lyric !== "string") throw new Error("failed");
    if (result.lyric.length > 51200) throw new Error("failed");
    return {
      lyric: result.lyric,
      tlyric: typeof result.tlyric === "string" && result.tlyric.length < 5120 ? result.tlyric : null,
      rlyric: typeof result.rlyric === "string" && result.rlyric.length < 5120 ? result.rlyric : null,
      lxlyric: typeof result.lxlyric === "string" && result.lxlyric.length < 8192 ? result.lxlyric : null,
    };
  }
  return result;
}

// pickHandler prefers the on(request) handler and falls back to a script that
// exports the action directly, which older sources do.
function pickHandler(ns, action) {
  if (typeof requestHandler === "function") return requestHandler;
  if (ns && typeof ns[action] === "function") return function (o) { return ns[action](o.info); };
  return null;
}

// One process per source, kept alive between requests. The desktop app runs a
// source in a WebView that outlives every song, so lx.send(inited) runs once;
// one node per request used to pay the init cost every time, which for a source
// that fetches its own config or account pool on init was the larger half of a
// lookup. loadOnce memoises the load so a crash cannot leave a half-initialised
// module behind, and drops the memo on failure so a transient load error cannot
// wedge the whole session. inFlight is what lets the process exit by itself
// once its pipe closes with nothing left to do.
var scriptLoad = null;
function loadOnce(p) {
  if (!scriptLoad) {
    // A failed load is not memoised: a script that fetches its config or key
    // list on load can fail once on a network blip, and caching that rejection
    // would turn a transient error into a session that never answers.
    scriptLoad = Promise.resolve().then(function () { return loadScript(p); })
      .catch(function (e) { scriptLoad = null; throw e; });
  }
  return scriptLoad;
}
var inFlight = 0;
var stdinClosed = false;
function maybeExit() {
  // Not process.exit: that truncates stdout and would eat the reply this very
  // call just wrote. exitCode lets node flush it and drain its own handles; Go
  // kills a source that lingers past that anyway.
  if (stdinClosed && inFlight === 0) process.exitCode = 0;
}
function handle(req) {
  inFlight++;
  var seconds = typeof req.timeoutSec === "number" && req.timeoutSec > 0 ? req.timeoutSec : 25;
  var answered = false;
  var kill = setTimeout(function () { say({ ok: false, error: "source did not answer in time" }); }, seconds * 1000);
  function say(v) {
    if (answered) return;
    answered = true;
    clearTimeout(kill);
    if (noTimer) clearTimeout(noTimer);
    reply(v);
    inFlight--;
    maybeExit();
  }

  // On a warm process the script has already announced itself, so inited is
  // settled and there is nothing to race it against. Arming the timeout on a
  // request that cannot use it would leave a timer pending past the request,
  // and this process outlives every song.
  var waitInit;
  var waitInitReject;
  var noTimer;
  if (!initedDone) {
    waitInit = new Promise(function (_, no) {
      waitInitReject = no;
      noTimer = setTimeout(function () {
        waitInitReject(new Error("source never called lx.send(inited): it declares no actions"));
      }, seconds * 1000);
    });
  }
  var gate = waitInit ? Promise.race([inited, waitInit]) : inited;

  // Describe mode only wants the init payload, which a script sends while it
  // loads. It issues no network request of its own, so it is fast and it works
  // even when the source's upstream backend is down.
  if (req.action === "__describe__") {
    loadOnce(process.argv[2])
      .then(function (ns) { return gate.then(function () { return ns; }); })
      .then(function () { say({ ok: true, result: null, init: initPayload }); })
      .catch(function (e) { say({ ok: false, error: errText(e) + failNote(), init: initPayload }); });
    return;
  }

  var info = clampQuality(req.action, req.info || {});

  loadOnce(process.argv[2])
    .then(function (ns) { return gate.then(function () { return ns; }); })
    .then(function (ns) {
      var fn = pickHandler(ns, req.action);
      if (!fn) {
        var exported = ns ? Object.keys(ns).join(", ") : "none";
        throw new Error("source exposed no handler for " + req.action + " (exports: " + exported + ")");
      }
      return fn({ source: info.source, action: req.action, info: info });
    })
    .then(function (result) {
      var out = validateResult(req.action, result);
      say({ ok: true, result: out === undefined ? null : out, init: initPayload, usedType: info.type });
    })
    .catch(function (e) { say({ ok: false, error: errText(e) + failNote(), init: initPayload }); });
}
// One request per line; the pipe stays open for the life of the process.
var stdinBuf = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", function (chunk) {
  stdinBuf += chunk;
  var i;
  while ((i = stdinBuf.indexOf("\n")) >= 0) {
    var line = stdinBuf.substring(0, i);
    stdinBuf = stdinBuf.substring(i + 1);
    if (!line.trim()) continue;
    var req;
    try { req = JSON.parse(line); } catch (e) { reply({ ok: false, error: "bad request json: " + e.message }); continue; }
    handle(req);
  }
});
process.stdin.on("end", function () { stdinClosed = true; maybeExit(); });
process.stdin.resume();
`

// ScriptInfo is the header block the app reads with parseScriptInfo: the script
// must begin with a /* @name ... @version ... */ comment or it is rejected.
type ScriptInfo struct {
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// UserSourceSpec is one configured user source. URL may be an http(s) URL or a
// local file path.
type UserSourceSpec struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Source      string `json:"source,omitempty"`     // platform whose songs it serves
	Node        string `json:"node,omitempty"`       // node executable
	TimeoutSec  int    `json:"timeoutSec,omitempty"` // per-request limit, 0 = 25
	CacheTTLHrs int    `json:"cacheTTLHours,omitempty"`
}

// SourceCapability is one platform a user source declares in its init payload.
// The preload filters the declared actions against ["musicUrl"] and the
// declared qualities against ["128k","320k","flac","flac24bit"], so a source
// that asks for flac may only deliver 128k.
type SourceCapability struct {
	Type     string   `json:"type"`
	Actions  []string `json:"actions"`
	Qualitys []string `json:"qualitys"`
}

// Config is the on-disk config, holding any number of user sources.
type Config struct {
	Node    string           `json:"node,omitempty"`
	Sources []UserSourceSpec `json:"sources"`
}

func (c *Config) Set(s UserSourceSpec) error {
	if c == nil {
		return fmt.Errorf("no config path available")
	}
	if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.URL) == "" {
		return fmt.Errorf("a source needs both a name and a url")
	}
	for i, old := range c.Sources {
		if old.Name == s.Name {
			c.Sources[i] = s
			return c.Save()
		}
	}
	c.Sources = append(c.Sources, s)
	return c.Save()
}

func (c *Config) Remove(name string) error {
	if c == nil {
		return fmt.Errorf("no config path available")
	}
	for i, s := range c.Sources {
		if s.Name == name {
			c.Sources = append(c.Sources[:i], c.Sources[i+1:]...)
			return c.Save()
		}
	}
	return fmt.Errorf("no source named %q", name)
}

func (c *Config) Spec(name string) (UserSourceSpec, bool) {
	if c == nil {
		return UserSourceSpec{}, false
	}
	for _, s := range c.Sources {
		if s.Name == name {
			return s, true
		}
	}
	return UserSourceSpec{}, false
}

// ConfigDir is where lxmusic keeps its config and its script cache.
func ConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if p := os.Getenv("APPDATA"); p != "" {
			return filepath.Join(p, "lxmusic-cli"), nil
		}
	case "darwin":
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, "Library", "Application Support", "lxmusic-cli"), nil
		}
	default:
		if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
			return filepath.Join(x, "lxmusic-cli"), nil
		}
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, ".config", "lxmusic-cli"), nil
		}
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".lxmusic-cli"), nil
}

// LoadConfig reads the config, creating an empty one when it does not exist.
func LoadConfig() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	c := &Config{}
	b, err := os.ReadFile(filepath.Join(dir, "lxmusic.json"))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("bad config %s: %v", filepath.Join(dir, "lxmusic.json"), err)
	}
	return c, nil
}

func (c *Config) Save() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "lxmusic.json"), b, 0o644)
}

// UserSource is a Provider backed by a user-provided script.
type UserSource struct {
	C    *Client
	Spec UserSourceSpec
	Info ScriptInfo
	Cap  map[string]SourceCapability // learned from the init payload, per platform

	// sess is the node process serving this source's script for the whole run,
	// so the script's init runs once instead of once per request. sessMu covers
	// it and serialises requests, because one process cannot serve two at once.
	sessMu sync.Mutex
	sess   *bridgeSession
}

// Capable reports the platform a song belongs to, or nil if the source never
// announced itself.
func (u *UserSource) Capable(s *Song) *SourceCapability {
	src := s.Source
	if src == "" {
		src = u.Spec.Source
	}
	if c, ok := u.Cap[strings.ToLower(src)]; ok {
		return &c
	}
	return nil
}

// Require fails when the source declared that it cannot do an action. A source
// that never announced itself is let through so it can still try: some scripts
// register their handlers lazily.
func (u *UserSource) Require(s *Song, action string) error {
	c := u.Capable(s)
	if c == nil {
		return nil
	}
	if contains(c.Actions, action) {
		return nil
	}
	return fmt.Errorf("source %q does not serve %q (declared actions: %s)",
		u.Spec.Name, action, strings.Join(c.Actions, ", "))
}

func (u *UserSource) ID() string { return u.Spec.Name }

// Name is the display name shown by "lxmusic platforms" and the search table.
// u.Info.Name is the name the script announced at init, but that only exists
// after the script has been fetched and run — Describe or the first request.
// Falling back to the config id means a freshly added source prints something
// readable instead of a blank column, without waiting for a node round trip.
func (u *UserSource) Name() string {
	if u.Info.Name != "" {
		return u.Info.Name
	}
	return u.Spec.Name
}

// Search is not part of the user source contract: the app never searches through
// a user source, it resolves the song on a platform source first and only asks
// the user source for the url / lyric / pic.
func (u *UserSource) Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error) {
	return nil, fmt.Errorf("%q: %w: resolve the song on a platform source first, then pick it with --source %s",
		u.Spec.Name, ErrNoSearch, u.Spec.Name)
}

// Detail is a no-op: the platform search already carries the metadata.
func (u *UserSource) Detail(ctx context.Context, s *Song) error { return nil }

// Anonymous audio: this is the point of a user source - it is the adapter that
// supplies the audio url the built-in providers will not. Search is never part
// of it, so a user source never puts rows into the search table.
func (u *UserSource) AnonymousAudio() bool { return true }

// Describe loads the script and reports what it declared: the platforms,
// actions and qualities it can serve. That is the ground truth behind quality
// selection, since a source offering only 128k will never deliver flac however
// the CLI is asked. It makes no upstream request, so it still works when the
// source's backend is down.
func (u *UserSource) Describe(ctx context.Context) error {
	_, _, err := u.call(ctx, "__describe__", map[string]any{})
	return err
}

// ensureScript fetches or reads the script, keeping it in the cache.
func (u *UserSource) ensureScript(ctx context.Context, refresh bool) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	cache := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return "", err
	}
	if err := writeBridge(cache); err != nil {
		return "", err
	}
	dest := filepath.Join(cache, u.Spec.Name+".js")

	if !refresh && !localPath(u.Spec.URL) {
		if st, err := os.Stat(dest); err == nil && time.Since(st.ModTime()) < time.Duration(u.cacheTTLHrs())*time.Hour {
			return dest, nil
		}
	}
	var raw []byte
	if localPath(u.Spec.URL) {
		raw, err = os.ReadFile(u.Spec.URL)
	} else {
		raw, err = u.C.GetBytes(ctx, u.Spec.URL, nil)
	}
	if err != nil {
		return "", fmt.Errorf("source %q: %v", u.Spec.Name, err)
	}
	if info, ok := parseScriptInfo(string(raw)); ok {
		u.Info = info
	}
	if u.Info.Name == "" {
		return "", fmt.Errorf("source %q: invalid user source file: it must start with a /* @name ... @version */ block", u.Spec.Name)
	}
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}

// writeBridge installs the node-side loader. It is checked on every call
// because it ships with the CLI, not with the script: the script may be a day
// old while the bridge is brand new.
func writeBridge(cache string) error {
	bridge := filepath.Join(cache, "bridge.js")
	return writeIfChanged(bridge, userBridge)
}

// writeIfChanged rewrites p only when its content differs from want. The bridge
// is regenerated as the CLI is rebuilt; rewriting a large file on every call
// would be the wasteful half of that.
func writeIfChanged(path, want string) error {
	if cur, err := os.ReadFile(path); err == nil && string(cur) == want {
		return nil
	}
	return os.WriteFile(path, []byte(want), 0o644)
}

func (u *UserSource) cacheTTLHrs() int {
	if u.Spec.CacheTTLHrs > 0 {
		return u.Spec.CacheTTLHrs
	}
	return 24
}

func (u *UserSource) timeout() time.Duration {
	if u.Spec.TimeoutSec > 0 {
		return time.Duration(u.Spec.TimeoutSec) * time.Second
	}
	return 25 * time.Second
}

func localPath(u string) bool {
	// Ask the filesystem before url.Parse: a Windows absolute path like "C:\foo"
	// parses as scheme "c", so url.Parse alone would misread every local path on
	// this platform as a URL and send it to the HTTP client.
	if filepath.IsAbs(u) {
		return true
	}
	p, err := url.Parse(u)
	if err != nil {
		return true
	}
	return p.Scheme == "" || p.Scheme == "file"
}

// parseScriptInfo mirrors the app's parseScriptInfo, which requires a leading
// block comment and truncates each field the way INFO_NAMES does.
func parseScriptInfo(script string) (ScriptInfo, bool) {
	if !strings.HasPrefix(strings.TrimSpace(script), "/*") {
		return ScriptInfo{}, false
	}
	m := regexp.MustCompile(`^/\*[\s\S]+?\*/`).FindString(script)
	if m == "" {
		return ScriptInfo{}, false
	}
	out := ScriptInfo{}
	for _, line := range strings.Split(m, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "* "))
		switch {
		case strings.HasPrefix(line, "@name"):
			out.Name = strings.TrimSpace(strings.TrimPrefix(line, "@name"))
			out.Name = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(out.Name, ":"), "-"))
		case strings.HasPrefix(line, "@version"):
			out.Version = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "@version"), ":"))
		case strings.HasPrefix(line, "@description"):
			out.Description = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "@description"), ":"))
		}
	}
	return out, true
}

// songInfo builds the info object the app hands to a user source. renderer.js
// sends info: { type, musicInfo }, where musicInfo is the song and type is the
// requested quality for musicUrl but the source type "music" for lyric and pic.
// The flat fields are repeated at the top level because not every published
// source reads the nested shape.
func songInfo(s *Song, fallbackSource, quality, action string) map[string]any {
	qs := make([]string, 0, len(s.Qualities))
	for _, q := range s.Qualities {
		qs = append(qs, q.Type)
	}
	mid := s.MID
	if mid == "" {
		mid = s.ID
	}
	src := s.Source
	if src == "" {
		src = fallbackSource
	}
	if action != "musicUrl" {
		quality = "music"
	}
	musicInfo := map[string]any{
		"source":    src,
		"name":      s.Name,
		"singer":    s.Singer,
		"album":     s.Album,
		"albumName": s.Album,
		"albumId":   s.AlbumID,
		"songmid":   mid,
		"mid":       mid,
		"id":        s.ID,
		"songId":    s.NumericID,
		"duration":  s.Duration,
		"interval":  s.Duration,
		"types":     qs,
		"type":      quality,
		"img":       s.Pic,
		"sourceUrl": s.PageURL,
	}
	// hash is left out rather than set to "" when the platform has no hash: a
	// source resolving the id as `musicInfo.hash ?? musicInfo.songmid` would
	// otherwise bind to the empty string (?? only skips null and undefined) and
	// hit the upstream with an empty path segment. Dropping the key lets it fall
	// through to songmid, which carries kuwo's numeric id.
	if s.Hash != "" {
		musicInfo["hash"] = s.Hash
	}
	out := make(map[string]any, len(musicInfo)+1)
	for k, v := range musicInfo {
		out[k] = v
	}
	out["musicInfo"] = musicInfo
	return out
}

// bridgeReply is one line of the bridge's protocol.
type bridgeReply struct {
	OK       bool            `json:"ok"`
	Result   json.RawMessage `json:"result"`
	Error    string          `json:"error"`
	Init     json.RawMessage `json:"init"`
	UsedType string          `json:"usedType"`
}

// apply turns a reply into a result. The second return value is the quality the
// script was actually asked for: the bridge settles an unknown or unsupported
// request on the highest quality the script declared, which only it can see, so
// Go must not decide this itself.
func (u *UserSource) apply(r bridgeReply) (any, string, error) {
	if len(r.Init) > 0 {
		var d struct {
			Sources map[string]SourceCapability `json:"sources"`
		}
		if json.Unmarshal(r.Init, &d) == nil {
			u.Cap = d.Sources
		}
	}
	if !r.OK {
		return nil, "", fmt.Errorf("source %q: %s", u.Spec.Name, r.Error)
	}
	var out any
	if len(r.Result) > 0 {
		if err := json.Unmarshal(r.Result, &out); err != nil {
			return nil, "", fmt.Errorf("source %q: bad result: %v", u.Spec.Name, err)
		}
	}
	return out, r.UsedType, nil
}

// call runs one action against the script in node.
func (u *UserSource) call(ctx context.Context, action string, info map[string]any) (any, string, error) {
	script, err := u.ensureScript(ctx, false)
	if err != nil {
		return nil, "", err
	}
	payload := map[string]any{
		"action":     action,
		"info":       info,
		"timeoutSec": int(u.timeout().Seconds()),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	b = append(b, '\n') // the bridge reads requests line by line
	cctx, cancel := context.WithTimeout(ctx, u.timeout()+5*time.Second)
	defer cancel()

	// A live session reuses the script's inited state, which is most of a
	// request's cost, so it always goes first. A broken pipe means the process
	// died - usually a source that exits during init - so it is dropped and one
	// fresh attempt is made; if that cannot be brought up either the request
	// falls back to a disposable process, which is what this did before
	// sessions existed. The session is only trusted after it has answered, so a
	// process that starts and immediately fails is never kept.
	u.sessMu.Lock()
	defer u.sessMu.Unlock()
	var lastErr error
	for i := 0; i < 2; i++ {
		s := u.sess // the whole point: reuse the process that already inited
		if s == nil {
			s = u.spawn(script)
		}
		if s == nil {
			break
		}
		r, err := s.do(cctx, b)
		if err == nil {
			u.sess = s
			return u.apply(r)
		}
		lastErr = err
		u.sess = nil
		s.close()
		if cctx.Err() != nil {
			// The caller's deadline decided, not a broken pipe: there is no
			// point starting another node to meet it.
			return nil, "", lastErr
		}
	}
	return u.callOnce(cctx, script, b)
}

// callOnce serves one request to a disposable process. It is the fallback for a
// source that will not cooperate with a long-lived pipe.
func (u *UserSource) callOnce(ctx context.Context, script string, payload []byte) (any, string, error) {
	node, err := u.nodePath()
	if err != nil {
		return nil, "", err
	}
	sink := &errSink{}
	cmd := exec.CommandContext(ctx, node, filepath.Join(filepath.Dir(script), "bridge.js"), script)
	cmd.Stdin = bytes.NewReader(payload)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	cmd.Stderr = sink
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("source %q: node failed: %v", u.Spec.Name, err)
	}
	line, err := bufio.NewReader(stdout).ReadBytes('\n')
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, "", fmt.Errorf("source %q: node failed: %v%s", u.Spec.Name, err, snippet(sink.tail()))
	}
	var r bridgeReply
	if err := json.Unmarshal(bytes.TrimSpace(line), &r); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, "", fmt.Errorf("source %q: bad response %q%s", u.Spec.Name, snippet(string(line)), snippet(sink.tail()))
	}
	// The reply is out; a source may still be holding sockets it opened, so do
	// not wait for it to drain on its own.
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return u.apply(r)
}

// bridgeSession is one node process serving a source's script for the whole
// run. Requests are serialised by UserSource.sessMu because the bridge answers
// one at a time; replies are single lines, so a pipe reader is enough framing.
type bridgeSession struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	stderr    *errSink
	closeOnce sync.Once
}

func (s *bridgeSession) close() {
	s.closeOnce.Do(func() {
		_ = s.stdin.Close()
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		_ = s.cmd.Wait()
	})
}

// do writes one request and waits for its reply, bounded by ctx. On timeout the
// caller is expected to close the session, which breaks the blocking read.
func (s *bridgeSession) do(ctx context.Context, payload []byte) (bridgeReply, error) {
	type outcome struct {
		r   bridgeReply
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		if _, err := s.stdin.Write(payload); err != nil {
			ch <- outcome{err: err}
			return
		}
		line, err := s.stdout.ReadBytes('\n')
		if err != nil {
			ch <- outcome{err: err}
			return
		}
		var r bridgeReply
		if err := json.Unmarshal(bytes.TrimSpace(line), &r); err != nil {
			ch <- outcome{err: err}
			return
		}
		ch <- outcome{r: r}
	}()
	select {
	case o := <-ch:
		return o.r, o.err
	case <-ctx.Done():
		return bridgeReply{}, ctx.Err()
	}
}

// spawn starts a long-lived node for the source, or returns nil when node is
// not available at all. The caller holds u.sessMu.
func (u *UserSource) spawn(script string) *bridgeSession {
	node, err := u.nodePath()
	if err != nil {
		return nil
	}
	cmd := exec.Command(node, filepath.Join(filepath.Dir(script), "bridge.js"), script)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil
	}
	s := &bridgeSession{stderr: &errSink{}}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		return nil
	}
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReader(stdout)
	return s
}

// nodePath resolves the node executable, with the config override first.
func (u *UserSource) nodePath() (string, error) {
	if u.Spec.Node != "" {
		return u.Spec.Node, nil
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("source %q needs node on PATH to run its script (set \"node\" in the config)", u.Spec.Name)
	}
	return node, nil
}

// errSink is stderr for a process that may outlive the request that produced
// it. It keeps only the tail, so a chatty source cannot grow it without bound,
// and it is safe for the read goroutine and the caller to touch at once.
type errSink struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (e *errSink) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.b.Len() > 4096 {
		src := e.b.Bytes()
		copy(src[:0], src[len(src)-2048:])
		e.b.Truncate(2048)
	}
	return e.b.Write(p)
}

func (e *errSink) tail() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.b.String()
}

func snippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		s = s[:400] + "..."
	}
	if s == "" {
		return ""
	}
	return ": " + s
}

// note attaches stderr without the colon prefix snippet uses, so a source that
// dumps its own error there is visible when the reply never arrives.
func note(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		s = s[:400] + "..."
	}
	if s == "" {
		return ""
	}
	return " (stderr: " + s + ")"
}

// MusicURL asks the script for the audio URL. Go supplies only the user's
// preference; the bridge settles it on the highest quality the script declared,
// because the script's own init payload is the ground truth and it is not
// visible to Go until this call has answered.
func (u *UserSource) MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error) {
	if err := u.Require(s, "musicUrl"); err != nil {
		return nil, err
	}
	t := quality.Type
	if t == "" || t == QAuto || t == QBestKnown {
		t = s.BestQuality()
		if t == "" {
			// No quality matrix - a song known only by id. The script is the only
			// thing that can see its own capability table, so ask it rather than
			// assume the lowest tier: that assumption is what made "auto" come
			// back as a 2.8 MB ogg while the search table advertised flac.
			if q := u.bestDeclared(ctx, s); q != "" {
				t = q
			}
		}
	}
	if t == "" {
		t = Q128
	}
	out, used, err := u.call(ctx, "musicUrl", songInfo(s, u.Spec.Source, t, "musicUrl"))
	if err != nil {
		return nil, err
	}
	if err := userAPIError(out); err != nil {
		return nil, err
	}
	streamURL, q := extractURL(out, t)
	if streamURL == "" {
		return nil, fmt.Errorf("source %q returned no url for %s", u.Spec.Name, s.KeyString())
	}
	// The url wins if the script names the quality it handed back; otherwise the
	// quality it was actually asked for, which is the clamped one, not t.
	if q == "" {
		q = used
	}
	if q == "" {
		q = t
	}
	headers := map[string]string{}
	if s.PageURL != "" {
		headers["Referer"] = s.PageURL
	}
	return &PlayerURL{
		URL: streamURL, Quality: q, Source: u.Spec.Name, Duration: s.Duration,
		FileName: s.Name + AudioExt(q, streamURL),
		Headers:  headers,
	}, nil
}

// bestDeclared is the highest quality tier the script announces for the song's
// platform, or "" when it cannot be determined. It costs a script init, which is
// why it is only called when the song carries no quality matrix of its own, and
// the answer is remembered in Cap for the rest of the process.
func (u *UserSource) bestDeclared(ctx context.Context, s *Song) string {
	if len(u.Cap) == 0 {
		if err := u.Describe(ctx); err != nil {
			return ""
		}
	}
	if c := u.Capable(s); c != nil {
		return bestOf(c.Qualitys)
	}
	return ""
}

// Lyric asks the script for the lyric.
func (u *UserSource) Lyric(ctx context.Context, s *Song) (*Lyric, error) {
	if err := u.Require(s, "lyric"); err != nil {
		return nil, err
	}
	out, _, err := u.call(ctx, "lyric", songInfo(s, u.Spec.Source, "", "lyric"))
	if err != nil {
		return nil, err
	}
	if err := userAPIError(out); err != nil {
		return nil, err
	}
	plain, trans, roman, format := extractLyric(out)
	if strings.TrimSpace(plain) == "" {
		return nil, fmt.Errorf("source %q returned no lyric for %s", u.Spec.Name, s.KeyString())
	}
	if format == "" {
		format = "lrc"
	}
	return &Lyric{
		Plain: strings.TrimSpace(plain), Trans: strings.TrimSpace(trans),
		Roman: strings.TrimSpace(roman), Format: format,
	}, nil
}

// Pic asks the script for the cover.
func (u *UserSource) Pic(ctx context.Context, s *Song) (string, error) {
	if err := u.Require(s, "pic"); err != nil {
		return "", err
	}
	out, _, err := u.call(ctx, "pic", songInfo(s, u.Spec.Source, "", "pic"))
	if err != nil {
		return "", err
	}
	if err := userAPIError(out); err != nil {
		return "", err
	}
	if v := extractString(out, "url", "pic", "img", "src", "data"); v != "" {
		return v, nil
	}
	if s, ok := out.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("source %q returned no pic for %s", u.Spec.Name, s.KeyString())
}

// userAPIError surfaces the { code: -1, msg: "..." } failure shape.
func userAPIError(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	if c, ok := m["code"]; ok {
		switch n := c.(type) {
		case float64:
			if n != 0 {
				return fmt.Errorf("source failed: %s", extractString(m, "msg", "message", "error"))
			}
		case string:
			if n != "0" {
				return fmt.Errorf("source failed: %s", extractString(m, "msg", "message", "error"))
			}
		}
	}
	if msg := extractString(m, "msg", "message"); msg != "" && extractString(m, "url", "lyric", "pic") == "" {
		if strings.Contains(strings.ToLower(msg), "fail") {
			return fmt.Errorf("source failed: %s", msg)
		}
	}
	return nil
}

// extractString digs the first named field out of nested objects.
func extractString(v any, names ...string) string {
	switch x := v.(type) {
	case string:
		if strings.TrimSpace(x) != "" {
			return x
		}
	case map[string]any:
		if c, ok := x["code"]; ok {
			if n, ok := c.(float64); ok && n != 0 {
				return ""
			}
		}
		for _, n := range names {
			if s, ok := x[n].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
		if s := extractString(x["data"], names...); s != "" {
			return s
		}
	}
	return ""
}

// extractURL pulls a stream URL out of whatever shape the source returned.
func extractURL(v any, wantQuality string) (url, quality string) {
	m, ok := v.(map[string]any)
	if !ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s), ""
		}
		return "", ""
	}
	if c, ok := m["code"]; ok {
		if n, ok := c.(float64); ok && n != 0 {
			return "", ""
		}
	}
	if q := extractString(m, "quality", "bitrate", "type"); q != "" {
		quality = q
	}
	// A per-quality map: { "128k": "...", "320k": "..." }.
	if d, ok := m["data"].(map[string]any); ok {
		if want := strings.ToLower(wantQuality); want != "" {
			if u, ok := d[want].(string); ok && u != "" {
				return u, want
			}
		}
		for _, k := range []string{wantQuality, "flac", "320k", "128k"} {
			if u, ok := d[strings.ToLower(k)].(string); ok && u != "" {
				return u, strings.ToLower(k)
			}
		}
	}
	if u := extractString(m, "url", "src", "link", "href", "data"); u != "" {
		return u, quality
	}
	return "", ""
}

// extractLyric pulls the plain / translation / romanization out of the result.
// A source may also return lxlyric, kuwo's per-word timed variant, which parses
// into a richer lyric and wins over a plain lrc when it does.
func extractLyric(v any) (plain, trans, roman, format string) {
	if s, ok := v.(string); ok {
		return s, "", "", "lrc"
	}
	plain = extractString(v, "lyric", "lrc", "plain")
	trans = extractString(v, "tlyric", "trans", "translation")
	roman = extractString(v, "rlyric", "roma", "romanization")
	format = extractString(v, "format", "kind")
	if plain == "" {
		plain = extractString(v, "data")
	}
	if raw := extractString(v, "lxlyric"); raw != "" {
		if ly := parseKuwoLRC(raw); ly != nil && strings.TrimSpace(ly.Plain) != "" {
			plain = ly.Plain
			if strings.TrimSpace(trans) == "" {
				trans = ly.Trans
			}
			format = "lrc"
		}
	}
	if format == "" {
		format = "lrc"
	}
	return
}
