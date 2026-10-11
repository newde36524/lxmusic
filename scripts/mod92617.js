// Sacrificial first line.
// Find every occurrence of the webpack module id 92617 and dump the definition.
const fs = require('fs');
const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
let i = -1, out = [];
while ((i = src.indexOf('92617', i + 1)) >= 0) out.push(i);
console.log('occurrences:', out.length);
out.forEach((idx, n) => {
  const w = src.substring(idx - 200, idx + 200);
  console.log('---', n, idx, JSON.stringify(w.replace(/\n/g, ' ').slice(0, 200)));
});
const def = out.find(idx => src.substring(idx + 6, idx + 22).includes('function')) || out[out.length - 1];
if (def >= 0) {
  const start = src.indexOf('{', src.indexOf('=>', def));
  const w = src.substring(def, def + 12000);
  fs.writeFileSync('C:/lx-rl/probe/mod.js', w.replace(/;/g, ';\n').replace(/,/g, ',\n'));
  console.log('dumped def at', def);
}
