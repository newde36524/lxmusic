// usage: node win.js <file> <anchor> <radiusChars> <maxHits>
const fs = require("fs");
const file = process.argv[2], anchor = process.argv[3];
const radius = +(process.argv[4] || 6000), maxHits = +(process.argv[5] || 3);
const text = fs.readFileSync(file, "utf8");
let i = 0, n = 0;
while (n < maxHits) {
  const p = text.indexOf(anchor, i);
  if (p < 0) break;
  const s = Math.max(0, p - Math.floor(radius * 0.45)), e = Math.min(text.length, p + radius);
  console.log("\n===== HIT " + (n + 1) + " at offset " + p + " =====\n");
  console.log(text.slice(s, e));
  i = p + anchor.length;
  n++;
}
if (!n) console.log("NO HITS for: " + anchor);
