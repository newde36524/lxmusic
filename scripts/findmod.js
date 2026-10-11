// Sacrificial first line.
// Scan every dist chunk for the webpack module id given as argv[2].
const fs = require('fs');
const path = 'C:/lx-rl/extracted/dist/';
const id = process.argv[2] || '92617';
for (const f of fs.readdirSync(path)) {
  if (!f.endsWith('.js')) continue;
  const t = fs.readFileSync(path + f, 'utf8');
  const hits = [];
  let i = -1;
  while ((i = t.indexOf(id, i + 1)) >= 0) hits.push(i);
  if (hits.length) {
    console.log(f, t.length, hits.length);
    hits.forEach(h => console.log('   at', h, JSON.stringify(t.substring(h - 40, h + 120))));
  }
}
