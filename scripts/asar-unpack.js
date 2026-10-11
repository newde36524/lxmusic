// Minimal read-only asar extractor. Usage: node asar-unpack.js <app.asar> <outDir>
const fs = require('fs');
const path = require('path');

const [,, asarPath, outDir] = process.argv;
if (!asarPath || !outDir) {
  console.error('usage: node asar-unpack.js <app.asar> <outDir>');
  process.exit(1);
}

const buf = fs.readFileSync(asarPath);
// asar header: u32(4) | u32(pickle len) | u32(json len padded) | u32(json actual len) | json
if (buf.readUInt32LE(0) !== 4) throw new Error('not an asar archive');
const jsonLen = buf.readUInt32LE(12);
const base = 16 + ((jsonLen + 3) & ~3);
const header = JSON.parse(buf.toString('utf8', 16, 16 + jsonLen));

let files = 0, bytes = 0;
function walk(node, rel) {
  if (!node.files) {
    if (node.link) { // symlink: record as plain text so nothing executes
      fs.mkdirSync(path.dirname(path.join(outDir, rel)), { recursive: true });
      fs.writeFileSync(path.join(outDir, rel + '.LINK'), node.link);
      return;
    }
    const off = parseInt(node.offset, 10);
    const data = buf.slice(base + off, base + off + node.size);
    fs.mkdirSync(path.dirname(path.join(outDir, rel)), { recursive: true });
    fs.writeFileSync(path.join(outDir, rel), data);
    files++; bytes += data.length;
    return;
  }
  fs.mkdirSync(path.join(outDir, rel), { recursive: true });
  for (const [name, child] of Object.entries(node.files)) {
    walk(child, rel ? path.join(rel, name) : name);
  }
}
walk(header, '');
console.log(`ok: ${files} files, ${(bytes / 1048576).toFixed(1)} MB -> ${outDir}`);
