// Dump the three QQ lyric subkey matrices so the Go port can be compared row
// by row. Usage: txkeydump.js <outfile>
const fs = require("fs");
const { ne, ie } = require("C:/lx-rl/scripts/txjs.js");
const keys = [ne(ie.subarray(16, 24), 0), ne(ie.subarray(8, 16), 1), ne(ie.subarray(0, 8), 0)];
const lines = [];
keys.forEach((k, i) => {
  lines.push("key " + i);
  for (let r = 0; r < 16; r++) {
    lines.push(Buffer.from(k[r]).toString("hex"));
  }
  lines.push("");
});
fs.writeFileSync(process.argv[2], lines.join("\n"));
console.log("wrote " + process.argv[2] + " (" + lines.length + " lines)");
