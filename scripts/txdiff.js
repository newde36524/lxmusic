// Compare the Go port's post-cipher bytes against the JS reference.
// Usage: txdiff.js <hexfile> <jsbin> <gobin>
const fs = require("fs");
const { transform } = require("C:/lx-rl/scripts/txjs.js");
const hex = fs.readFileSync(process.argv[2], "utf8").trim();
const jsOut = transform(hex);
fs.writeFileSync(process.argv[3], jsOut);
const goOut = fs.readFileSync(process.argv[4]);
const n = Math.max(jsOut.length, goOut.length);
let first = -1;
for (let i = 0; i < n; i++) if (jsOut[i] !== goOut[i]) { first = i; break; }
console.log("js len " + jsOut.length + "  go len " + goOut.length + "  first diff @ " + first);
if (first >= 0) {
  const a = jsOut.subarray(Math.max(0, first - 4), first + 20), b = goOut.subarray(Math.max(0, first - 4), first + 20);
  console.log("js " + [...a].map((x) => x.toString(16).padStart(2, "0")).join(" "));
  console.log("go " + [...b].map((x) => x.toString(16).padStart(2, "0")).join(" "));
}
console.log("js head " + jsOut.subarray(0, 16).toString("hex"));
console.log("go head " + goOut.subarray(0, 16).toString("hex"));
