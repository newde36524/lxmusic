// Sacrificial first line.
// Dump the kugou KRC decoder region of renderer.js, char-index based.
const fs = require('fs');
const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
const center = parseInt(process.argv[2] || '1352637', 10);
const span = parseInt(process.argv[3] || '4000', 10);
const w = src.substring(center - span, center + span);
const pretty = w.replace(/;/g, ';\n').replace(/,/g, ',\n');
fs.writeFileSync('C:/lx-rl/probe/krc.js', pretty);
console.log('wrote', pretty.length, 'chars around', center);
