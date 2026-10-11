// Locate the KRC decoder: string-only operations.
const fs = require('fs');
const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
for (const n of ['zzc', 'FLaw', '^2tGQ61', 'krc1', 'get_res_privilege', 'musicUrl']) {
  console.log(JSON.stringify(n), src.indexOf(n));
}
