// Probe kuwo's anti-CDN gateway directly so the CLI can report what kuwo
// actually replied instead of a JSON decode error.
const https = require("https");
const http = require("http");
const targets = process.argv.slice(2);
const get = (u) => new Promise((resolve) => {
  const lib = u.startsWith("https:") ? https : http;
  const r = lib.get(u, { headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) kwplayer_ar_9.2.2.1", "Referer": "http://www.kuwo.cn/" }, timeout: 25000 }, (res) => {
    let b = "";
    res.setEncoding("utf8");
    res.on("data", (c) => { b += c; });
    res.on("end", () => resolve({ u: u, status: res.statusCode, ct: res.headers["content-type"], body: b }));
  });
  r.on("timeout", () => r.destroy(new Error("timeout")));
  r.on("error", (e) => resolve({ u: u, error: e.code + " " + e.message }));
});
(async () => {
  for (const u of targets) {
    const r = await get(u);
    console.log("URL    : " + r.u);
    if (r.error) { console.log("ERROR  : " + r.error); continue; }
    console.log("STATUS : " + r.status + "  content-type: " + (r.ct || ""));
    console.log("BODY   : " + r.body.substring(0, 400).replace(/\s+/g, " "));
    console.log("---");
  }
})();
