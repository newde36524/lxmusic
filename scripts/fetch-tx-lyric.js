// Fetch QQ's lyric blob for a numeric song id and save it as hex files.
// Usage: fetch-tx-lyric.js <songId> <outdir>
const fs = require("fs");
const https = require("https");
const songId = parseInt(process.argv[2] || "97773", 10);
const dir = process.argv[3] || "C:/lx-rl/scripts";
const body = JSON.stringify({
  comm: { ct: "19", cv: "1859", uin: "0" },
  req: { method: "GetPlayLyricInfo", module: "music.musichallSong.PlayLyricInfo",
    param: { format: "json", crypt: 1, ct: 19, cv: 1873, interval: 0, lrc_t: 0, qrc: 1, qrc_t: 0, roma: 1, roma_t: 0, songID: songId, trans: 1, trans_t: 0, type: -1 } },
});
https.request("https://u.y.qq.com/cgi-bin/musicu.fcg", {
  method: "POST", timeout: 30000,
  headers: { "content-type": "application/json", "content-length": Buffer.byteLength(body), "referer": "https://y.qq.com", "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36" },
}, (res) => {
  let b = "";
  res.setEncoding("utf8");
  res.on("data", (c) => { b += c; });
  res.on("end", () => {
    const j = JSON.parse(b);
    const d = (j.req || {}).data || {};
    console.log("code " + j.code + "/" + (j.req || {}).code);
    for (const k of ["lyric", "trans", "roma", "lrc", "qrc"]) {
      const v = d[k] || "";
      fs.writeFileSync(dir + "/" + k + ".hex", v);
      console.log("  " + k + " " + v.length);
    }
    console.log("keys: " + Object.keys(d).join(","));
  });
}).on("error", (e) => console.log("ERROR " + e.code + " " + e.message)).end(body);
