// Probe PlayLyricInfo with crypt:0 (and crypt:1 for comparison) to see whether
// QQ will hand back plain LRC without the cipher. Usage: txcrypt0.js <songId>
const https = require("https");
const songId = parseInt(process.argv[2] || "97773", 10);
const { decode } = require("C:/lx-rl/scripts/txjs.js");
const call = (crypt) =>
  new Promise((resolve) => {
    const body = JSON.stringify({
      comm: { ct: "19", cv: "1859", uin: "0" },
      req: { method: "GetPlayLyricInfo", module: "music.musichallSong.PlayLyricInfo",
        param: { format: "json", crypt, ct: 19, cv: 1873, interval: 0, lrc_t: 0,
          qrc: 1, qrc_t: 0, roma: 1, roma_t: 0, songID: songId, trans: 1, trans_t: 0, type: -1 } },
    });
    https.request("https://u.y.qq.com/cgi-bin/musicu.fcg", {
      method: "POST", timeout: 30000,
      headers: { "content-type": "application/json", "content-length": Buffer.byteLength(body),
        referer: "https://y.qq.com", "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36" },
    }, (res) => {
      let b = "";
      res.setEncoding("utf8");
      res.on("data", (c) => { b += c; });
      res.on("end", () => resolve(JSON.parse(b).req.data));
    }).on("error", () => resolve(null)).end(body);
  });
(async () => {
  for (const crypt of [0, 1]) {
    const d = await call(crypt) || {};
    console.log("=== crypt " + crypt + "  keys: " + Object.keys(d).join(","));
    const raw = d.lyric || "";
    console.log("lyric field length " + raw.length + "  looks-hex " + (/^[0-9a-f]+$/i.test(raw)));
    const out = crypt ? decode(raw) : raw;
    console.log(out.split("\n").slice(0, 14).join("\n"));
    console.log("");
  }
})();
