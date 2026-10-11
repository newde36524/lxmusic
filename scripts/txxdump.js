// Dump X(e,t,31-t) for t=0..63, to compare against the Go port bit by bit.
// Usage: txxdump.js <outfile>
const fs = require("fs");
const { X } = require("C:/lx-rl/scripts/txjs.js");
const e = Buffer.from([33, 64, 35, 41, 40, 78, 72, 76]);
const lines = [];
for (let t = 0; t < 64; t++) lines.push(t + " " + X(e, t, 31 - t).toString(16));
fs.writeFileSync(process.argv[2], lines.join("\n"));
console.log(lines.join(" "));
