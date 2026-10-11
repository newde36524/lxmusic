// Print a column window of one line of a file: col.js file line start end
const fs = require("fs");
const s = fs.readFileSync(process.argv[2], "utf8").split("\n");
const n = parseInt(process.argv[3], 10);
const from = parseInt(process.argv[4], 10);
const to = parseInt(process.argv[5], 10);
const line = s[n - 1] || "";
process.stdout.write(line.substring(from, to) + "\n");
