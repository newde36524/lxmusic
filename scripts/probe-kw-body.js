// Dump the start of the file kuwo's convert_url points at.
const https = require("https");
const http = require("http");
const rid = process.argv[2];
const get = (u, headers) => new Promise((resolve) => {
  const lib = u.startsWith("https:") ? https : http;
  const h = Object.assign({ "User-Agent": "Mozilla/5.0 kwplayer_ar_9.2.2.1", "Referer": "http://www.kuwo.cn/" }, headers || {});
  const r = lib.get(u, { headers: h, timeout: 30000 }, (res) => {
    let b = Buffer.alloc(0);
    res.on("data", (c) => { b = Buffer.concat([b, c]); });
    res.on("end", () => resolve({ status: res.statusCode, headers: res.headers, body: b }));
  });
  r.on("error", (e) => resolve({ error: e.code + " " + e.message }));
});
(async () => {
  const conv = await get("http://antiserver.kuwo.cn/anti.s?type=convert_url&rid=" + rid + "&format=mp3&br=128000");
  const u = conv.body.toString("utf8").trim();
  console.log("convert_url -> " + u);
  const mp3 = await get(u, { Range: "bytes=0-60000" });
  if (mp3.error) { console.log("ERROR " + mp3.error); return; }
  const b = mp3.body;
  console.log("status " + mp3.status + "  content-range " + (mp3.headers["content-range"] || "-") + "  bytes read " + b.length);
  console.log("content-type " + (mp3.headers["content-type"] || "-"));
  console.log("--- first 240 bytes (latin1) ---");
  console.log(JSON.stringify(b.slice(0, 240).toString("latin1")));
  console.log("--- magic ---");
  console.log("hex " + b.slice(0, 16).toString("hex"));
})();
