// Pull the QQ lyric cipher constants out of the app bundle as plain text. The
// bundle is only ever read; nothing in it is executed here.
const fs = require("fs");
const all = fs.readFileSync(process.argv[2], "utf8");
// Scope every search to the cipher region: `,K=[` and friends occur dozens of
// times elsewhere in the bundle.
const hStart = all.indexOf("const H=[[");
const src = all.substring(hStart, hStart + 8000);

// find an array literal whose '[' sits at brOpen and return its contents
function takeArray(brOpen) {
  let d = 0, i = brOpen, q = null;
  while (i < src.length) {
    const c = src[i];
    if (q) { if (c === "\\") i += 2; else if (c === q) q = null; else i++; continue; }
    if (c === '"' || c === "'" || c === "`") { q = c; i++; continue; }
    if (c === "[") d++; else if (c === "]") { d--; if (d === 0) return src.substring(brOpen + 1, i); }
    i++;
  }
  throw new Error("unbalanced at " + brOpen);
}

const lines = [];
for (const [name, needle] of [["H", "H=[["], ["K", ",K=["], ["G", ",G=["], ["W", ",W=["], ["Y", ",Y=["], ["ie", "ie=Buffer.from(["]]) {
  const at = src.indexOf(needle);
  if (at < 0) throw new Error("not found: " + needle);
  const br = src.indexOf("[", at);
  const inner = takeArray(br);
  const nums = inner.match(/-?\d+/g) || [];
  const groups = inner.match(/\[[^\]]*\]/g) || [];
  console.log(name + "  groups=" + (groups.length || 1) + "  nums=" + nums.length + "  head=" + JSON.stringify(nums.slice(0, 12)));
  lines.push("// " + name + "\n" + inner.replace(/\s+/g, " ").trim());
}
fs.writeFileSync(process.argv[3], lines.join("\n\n") + "\n");
console.log("wrote " + process.argv[3]);
