// Decode a QQ lyric hex blob with the cipher pulled out of the app bundle, so
// the Go port has ground truth to match. Usage: txdecode.js <hexfile>
const fs = require("fs");
const zlib = require("zlib");
const code = fs.readFileSync("C:/lx-rl/scripts/txcipher.js", "utf8");
// The cipher body uses `d.inflate` asynchronously; give it zlib and capture se.
const mod = "const Buffer=require('buffer').Buffer;\n" +
  "const d={inflate:(s,flag,cb)=>{try{cb(null,zlib.inflateSync(s,{flush:zlib.constants.Z_SYNC_FLUSH}))}catch(e){cb(e,null)}}," +
  "constants:{Z_SYNC_FLUSH:zlib.constants.Z_SYNC_FLUSH}};\n" +
  code + "\nmodule.exports={se,ie};";
fs.writeFileSync("C:/lx-rl/scripts/txcipher-mod.js", mod);
delete require.cache[require.resolve("C:/lx-rl/scripts/txcipher-mod.js")];
const { se } = require("C:/lx-rl/scripts/txcipher-mod.js");
const hex = fs.readFileSync(process.argv[2], "utf8").trim();
console.log("hex in : " + hex.length);
se(hex).then((out) => {
  console.log("decoded: " + out.length + " chars");
  fs.writeFileSync("C:/lx-rl/scripts/txlyric.out", out);
  console.log("--- head ---");
  console.log(out.split("\n").slice(0, 14).join("\n"));
});
