// Sacrificial first line.
// Dump the globalThis.lx surface that user-api-preload.js injects for user sources.
const fs = require("fs");
const file = process.argv[2] || "C:/lx-rl/extracted/dist/user-api-preload.js";
const s = fs.readFileSync(file, "utf8");
console.log("len=" + s.length);
const keys = process.argv.slice(3);
for (const key of keys) {
  let i = s.indexOf(key);
  let n = 0;
  while (i >= 0 && n < 4) {
    console.log("==== " + key + " at " + i + " ====");
    console.log(s.substring(Math.max(0, i - 320), i + 320).replace(/;/g, ";\n"));
    console.log();
    i = s.indexOf(key, i + 1);
    n++;
  }
}
