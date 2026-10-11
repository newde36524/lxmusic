// Dump the QQ key-schedule seed and each rotation, for the first key group.
// Usage: txseeddump.js <outfile>
const fs = require("fs");
const { X, K, G, W } = require("C:/lx-rl/scripts/txjs.js");
const e = Buffer.from([33, 64, 35, 41, 40, 78, 72, 76]);
let o = 0, i = 0;
for (let t = 0; t < 28; t++) o = (o | X(e, G[t], 31 - t)) >>> 0, i = (i | X(e, W[t], 31 - t)) >>> 0;
const lines = ["seed " + o.toString(16).padStart(8, "0") + " " + i.toString(16).padStart(8, "0")];
for (let r = 0; r < 16; r++) {
  o = (4294967280 & (o << K[r] >>> 0 | o >>> 28 - K[r])) >>> 0;
  i = (4294967280 & (i << K[r] >>> 0 | i >>> 28 - K[r])) >>> 0;
  lines.push("r" + r + " " + o.toString(16).padStart(8, "0") + " " + i.toString(16).padStart(8, "0"));
}
fs.writeFileSync(process.argv[2], lines.join("\n"));
console.log(lines[0]);
