// Dump what QQ's PlayLyricInfo actually returns, so the decoder matches the
// payload instead of the documentation.
const https = require("https");
const songId = parseInt(process.argv[2] || "216887536", 10);
const body = JSON.stringify({
  comm: { ct: 19, cv: 1859, uin: "0" },
  req: {
    method: "GetPlayLyricInfo",
    module: "music.musichallSong.PlayLyricInfo",
    param: { format: "json", crypt: 1, ct: 19, cv: 1873, interval: 0, lrc_t: 0, qrc: 1, qrc_t: 0, roma: 1, roma_t: 0, songID: songId, trans: 1, trans_t: 0, type: -1 },
  },
});
const req = https.request("https://u.y.qq.com/cgi-bin/musicu.fcg", {
  method: "POST", timeout: 30000,
  headers: { "content-type": "application/json", "content-length": Buffer.byteLength(body), "referer": "https://y.qq.com", "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36" },
}, (res) => {
  let b = "";
  res.setEncoding("utf8");
  res.on("data", (c) => { b += c; });
  res.on("end", () => {
    const j = JSON.parse(b);
    const d = (j.req && j.req.data) || {};
    for (const k of ["lyric", "trans", "roma"]) {
      const v = d[k] || "";
      console.log("=== " + k + "  len=" + v.length);
      if (!v) continue;
      console.log("first 120 raw : " + v.substring(0, 120));
      console.log("hex-ish?       : " + /^[0-9a-fA-F]+$/.test(v));
      console.log("base64-ish?    : " + /^[A-Za-z0-9+/=]+$/.test(v));
      try {
        const dec = Buffer.from(v, "base64");
        const xored = Buffer.from(dec.map((x) => x ^ 0xff));
        console.log("base64->xor0xff: " + JSON.stringify(xored.slice(0, 120).toString("utf8")));
      } catch (e) { console.log("base64 failed: " + e.message); }
    }
  });
});
req.on("timeout", () => req.destroy(new Error("timeout")));
req.on("error", (e) => console.log("ERROR " + e.code + " " + e.message));
req.write(body);
req.end();
