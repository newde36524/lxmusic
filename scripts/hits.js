// List every match offset + short context. Usage: node hits.js <file> <regex> [ctx]
const fs = require('fs');
const [,, file, reStr, ctxStr] = process.argv;
const ctx = ctxStr ? +ctxStr : 110;
const src = fs.readFileSync(file, 'utf8');
const re = new RegExp(reStr, 'g');
let m, n = 0;
while ((m = re.exec(src)) !== null) {
  const s = Math.max(0, m.index - ctx);
  console.log(`#${++n} @${m.index}: ...${src.slice(s, m.index + m[0].length + ctx)}...`);
  if (re.lastIndex === m.index) re.lastIndex++;
}
console.log(`total: ${n}`);
