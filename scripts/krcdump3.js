// Diagnose why the magic string is not found by node.
const fs = require('fs');
const p = 'C:/lx-rl/extracted/dist/renderer.js';
const buf = fs.readFileSync(p);
console.log('bytes', buf.length, 'first16hex', buf.slice(0, 16).toString('hex'));
const src = buf.toString('utf8');
console.log('utf8 chars', src.length);
for (const n of ['zzc', 'FLaw', '^2tGQ61', 'krc', 'get_res_privilege', 'musicUrl']) {
  console.log(JSON.stringify(n), 'utf8idx', src.indexOf(n), 'bufidx', buf.indexOf(n));
}
