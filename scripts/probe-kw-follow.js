// Ask kuwo for a convert_url, then measure the mp3 it points at, so the CLI
// does not advertise a bitrate the server will not actually stream.
const https = require("https");
const http = require("http");
const rids = process.argv.slice(2);

const get = (u, headers) => new Promise((resolve) => {
  const lib = u.startsWith("https:") ? https : http;
  const h = Object.assign({ "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) kwplayer_ar_9.2.2.1", "Referer": "http://www.kuwo.cn/" }, headers || {});
  const r = lib.get(u, { headers: h, timeout: 30000 }, (res) => {
    let b = Buffer.alloc(0);
    res.on("data", (c) => { b = Buffer.concat([b, c]); });
    res.on("end", () => resolve({ status: res.statusCode, headers: res.headers, body: b }));
  });
  r.on("timeout", () => r.destroy(new Error("timeout")));
  r.on("error", (e) => resolve({ error: e.code + " " + e.message }));
});

(async () => {
  for (const rid of rids) {
    console.log("=== rid " + rid);
    for (const br of ["128000", "320000"]) {
      const conv = await get("http://antiserver.kuwo.cn/anti.s?type=convert_url&rid=" + rid + "&format=mp3&br=" + br);
      if (conv.error) { console.log("  br=" + br + " ERROR " + conv.error); continue; }
      const u = conv.body.toString("utf8").trim();
      if (!/^https?:/.test(u)) { console.log("  br=" + br + " -> " + u); continue; }
      const mp3 = await get(u, { Range: "bytes=0-200000" });
      if (mp3.error) { console.log("  br=" + br + " mp3 ERROR " + mp3.error); continue; }
      const len = mp3.headers["content-length"];
      const rng = mp3.headers["content-range"];
      // The LAME / ID3 tag in the first frame tells us the real bitrate.
      const t = mp3.body.toString("latin1");
      const tag = t.match(/128 kbps|320 kbps|192 kbps|256 kbps|CBR 128|CBR 320/g);
      console.log("  br=" + br + " status=" + mp3.status + " range=" + (rng || "-") + " len=" + len + " tag=" + (tag ? tag[0] : "-"));
    }
  }
})();
