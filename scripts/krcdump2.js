// Locate the KRC decoder inside renderer.js and pretty-print it.
const fs = require('fs');
const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
const needle = process.argv[2] || 'FLaw';
const i = src.indexOf(needle);
console.log('needle', JSON.stringify(needle), 'idx', i, 'fileLen', src.length);
if (i >= 0) {
  const w = src.substring(Math.max(0, i - 7000), i + 9000);
  const pretty = w.replace(/;/g, ';\n').replace(/,/g, ',\n');
  fs.writeFileSync('C:/lx-rl/probe/krc.js', pretty);
  console.log('window', w.length, 'pretty', pretty.length);
}
