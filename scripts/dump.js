// Dump an absolute character range of a file, splitting at ';' for readability.
const fs = require("fs");
const s = fs.readFileSync(process.argv[2], "utf8");
const from = parseInt(process.argv[3], 10);
const to = parseInt(process.argv[4], 10);
process.stdout.write(s.substring(from, to).replace(/;/g, ";\n"));
