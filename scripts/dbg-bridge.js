
var fs = require("fs");
var Module = require("module");
var httpLib = require("http");
var httpsLib = require("https");
var crypto = require("crypto");
var path = require("path");

function doRequest(method, target, opts) { process.stderr.write("REQ [" + (method || "") + "] " + target + "\n");
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
        process.stderr.write("  <- " + res.statusCode + " " + buf.length + "B\n"); process.stderr.write("  body: " + buf.toString("utf8").substring(0, 320).replace(/\s+/g, " ") + "\n"); resolve(out);
      });
    });
    req.setTimeout(opts.timeout || 20000, function () { req.destroy(new Error("request timeout")); });
    req.on("error", function (e) { process.stderr.write('  ! ' + e.code + ' ' + e.message + "\n"); reject(e); });
    process.stderr.write("  hdrs: " + JSON.stringify(opts.headers || {}) + "\n"); process.stderr.write("  reqbody: " + String(body == null ? "" : body.toString("utf8")).substring(0, 300).replace(/\s+/g, " ") + "\n"); if (body != null && body !== "") req.write(body);
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

function readStdin() {
  return new Promise(function (resolve) {
    var d = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", function (c) { d += c; });
    process.stdin.on("end", function () { resolve(d); });
  });
}
function reply(v) { process.stdout.write(JSON.stringify(v)); }

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

lx.send = function (name, payload) {
  if (name === EVENT_NAMES.inited) {
    if (initPayload) return Promise.reject(new Error("Script is inited"));
    initPayload = payload;
    if (initedResolve) initedResolve(true);
    return Promise.resolve();
  }
  if (name === EVENT_NAMES.updateAlert) return Promise.resolve();
  return Promise.reject(new Error("The event is not supported: " + name));
};
lx.on = function (name, fn) { if (typeof fn === 'function') process.stderr.write('ON_HANDLER\n' + fn.toString() + '\n---END_HANDLER---\n');
  if (name === EVENT_NAMES.request) { requestHandler = fn; return Promise.resolve(); }
  return Promise.reject(new Error("The event is not supported: " + name));
};

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

readStdin().then(function (stdin) {
  var req;
  try { req = JSON.parse(stdin); } catch (e) { reply({ ok: false, error: "bad request json: " + e.message }); return; }
  var seconds = typeof req.timeoutSec === "number" && req.timeoutSec > 0 ? req.timeoutSec : 25;
  var answered = false;
  var kill = setTimeout(function () {
    if (!answered) { answered = true; reply({ ok: false, error: "source did not answer in time" }); }
  }, seconds * 1000);
  var say = function (v) { if (!answered) { answered = true; clearTimeout(kill); reply(v); } };

  var ns;
  try { ns = loadScript(process.argv[2]); }
  catch (e) { say({ ok: false, error: "failed to load source: " + (e && e.message || e) }); return; }

  var noInit = new Promise(function (_, no) {
    setTimeout(function () { no(new Error("source never called lx.send(inited): it declares no actions")); }, seconds * 1000);
  });

  Promise.resolve(ns)
    .then(function () { return Promise.race([inited, noInit]); })
    .then(function () {
      var fn = pickHandler(ns, req.action);
      if (!fn) {
        var exported = ns ? Object.keys(ns).join(", ") : "none";
        throw new Error("source exposed no handler for " + req.action + " (exports: " + exported + ")");
      }
      return fn({ source: req.info.source, action: req.action, info: req.info });
    })
    .then(function (result) {
      var out = validateResult(req.action, result);
      say({ ok: true, result: out === undefined ? null : out, init: initPayload });
    })
    .catch(function (e) { say({ ok: false, error: (e && e.stack ? e.stack : String(e)), init: initPayload }); });
});
