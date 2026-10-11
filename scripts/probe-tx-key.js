// Recover QQ's lyric cipher by brute force over single-byte XOR keys.
const https = require("https");
const songId = parseInt(process.argv[2] || "97773", 10);
const body = JSON.stringify({
  comm: { ct: 19, cv: 1859, uin: "0" },
  req: { method: "GetPlayLyricInfo", module: "music.musichallSong.PlayLyricInfo",
    param: { format: "json", crypt: 1, ct: 19, cv: 1873, interval: 0, lrc_t: 0, qrc: 1, qrc_t: 0, roma: 1, roma_t: 0, songID: songId, trans: 1, trans_t: 0, type: -1 } },
});
https.request("https://u.y.qq.com/cgi-bin/musicu.fcg", {
  method: "POST", timeout: 30000,
  headers: { "content-type": "application/json", "content-length": Buffer.byteLength(body), "referer": "https://y.qq.com", "User-Agent": "Mozilla/5.0 Chrome/86.0.4240.198 Safari/537.36" },
}, (res) => {
  let b = "";
  res.setEncoding("utf8");
  res.on("data", (c) => { b += c; });
  res.on("end", () => {
    const d = (JSON.parse(b).req || {}).data || {};
    for (const name of ["lyric", "trans", "roma"]) {
      const v = d[name] || "";
      if (!v) { console.log("=== " + name + " empty"); continue; }
      let buf = null;
      try { buf = Buffer.from(v, "hex"); } catch (e) {}
      if (!buf) { console.log("=== " + name + " not hex"); continue }
      console.log("=== " + name + "  hexlen=" + v.length + "  bytes=" + buf.length);
      console.log("    raw bytes (hex) : " + buf.slice(0, 16).toString("hex"));
      // score a single-byte key by how much printable/utf8 text it yields
      let best = [];
      for (let k = 0; k < 256; k++) {
        const out = Buffer.from(buf.map((x) => x ^ k));
        const s = out.toString("utf8");
        const good = (s.match(/[\x20-\x7e一-鿿\n\r[\]:.0-9]/g) || []).length;
        best.push([good, k]);
      }
      best.sort((a, b) => b[0] - a[0]);
      console.log("    top keys: " + best.slice(0, 5).map((x) => "0x" + x[1].toString(16) + "(" + x[0] + ")").join(" "));
      const k = best[0][1];
      const out = Buffer.from(buf.map((x) => x ^ k));
      console.log("    best 0x" + k.toString(16) + " -> " + JSON.stringify(out.slice(0, 160).toString("utf8")));
    }
  });
}).on("error", (e) => console.log("ERROR " + e.message)).end(body);
