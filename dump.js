const fs = require("fs");
const dir = process.argv[2];
fs.mkdirSync(dir, { recursive: true });
const cache = {}, idxMap = {}, namesMap = {};
for (const mf of fs.readdirSync("C:/lx-rl/extracted/dist").filter(f => f.endsWith(".js.map"))) {
  let map;
  try { map = JSON.parse(fs.readFileSync("C:/lx-rl/extracted/dist/" + mf, "utf8")); } catch (e) { continue; }
  cache[mf] = map;
  const idx = {};
  map.sources.forEach((s, i) => {
    let k = s.replace("webpack://lx-music-desktop/./", "").replace(/\?[0-9a-f]+$/, "");
    idx[k] = i;
  });
  idxMap[mf] = idx;
  namesMap[mf] = Object.keys(idx);
}
const wants = fs.readFileSync(0, "utf8").split(/\r?\n/).map(s => s.trim()).filter(Boolean);
const out = [];
for (const w of wants) {
  let hit = false;
  for (const mf in idxMap) {
    // match: exact base name, or base name + "?vue&type=" prefix
    const matches = namesMap[mf].filter(n => n === w || n.startsWith(w + "?vue&type="));
    for (const m of matches.sort()) {
      hit = true;
      const c = cache[mf].sourcesContent[idxMap[mf][m]];
      out.push("######## " + mf + " :: " + m + " (" + (c ? c.length : 0) + " chars)\n" + (c || "NO CONTENT"));
    }
  }
  if (!hit) out.push("######## NOT FOUND: " + w);
}
fs.writeFileSync(process.argv[3], out.join("\n\n"));
console.log("sections:", out.length, "bytes:", out.join("").length);
