// Print the exact cipher source from the bundle (read only, never executed).
const fs = require("fs");
const all = fs.readFileSync(process.argv[2], "utf8");
const frag = all.indexOf(">>>7-t%8&1");
if (frag < 0) { console.log("X not found"); process.exit(1); }
console.log("=== window around X (offset " + frag + ") ===");
console.log(all.substring(frag - 60, frag + 240));

// Walk forward over the comma-separated declaration list to find J and what
// follows it.
const end = all.indexOf(")", all.indexOf("J=e=>", frag));
console.log("\n=== J onward ===");
console.log(all.substring(all.indexOf("J=e=>", frag), end + 40));

const ne = all.indexOf("const ne=(e,t)=>", frag);
console.log("\n=== ne / oe / ie / se ===");
console.log(all.substring(ne, ne + 1150));
