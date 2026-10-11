// Sacrificial first line.
// Dump renderer.js[start, start+length], pretty-printed (one token per line).
const fs = require('fs');
const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
const start = parseInt(process.argv[2] || '0', 10);
const len = parseInt(process.argv[3] || '4000', 10);
const w = src.substring(start, start + len);
fs.writeFileSync('C:/lx-rl/probe/win.js', w.replace(/;/g, ';\n').replace(/,/g, ',\n'));
console.log('wrote', w.length);
