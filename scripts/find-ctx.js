// Find every occurrence of a pattern in a file and print a window around it.
// Usage: find-ctx.js <file> <regex> <window> [maxMatches]
const fs = require("fs");
const file = process.argv[2];
const re = new RegExp(process.argv[3]);
const win = parseInt(process.argv[4] || "220", 10);
const max = parseInt(process.argv[5] || "12", 10);
const s = fs.readFileSync(file, "utf8");
let n = 0, m;
const scan = new RegExp(re.source, "g" + (re.flags.includes("i") ? "i" : ""));
while ((m = scan.exec(s)) !== null && n < max) {
  n++;
  const from = Math.max(0, m.index - win);
  const to = Math.min(s.length, m.index + m[0].length + win);
  console.log("--- match " + n + " at " + m.index + "  " + JSON.stringify(m[0].substring(0, 120)));
  console.log(s.substring(from, to).replace(/\s+/g, " "));
  console.log("");
}
console.log("total matches shown: " + n);
