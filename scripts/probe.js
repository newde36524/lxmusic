// Print code context around regex matches in a minified bundle.
// Usage: node probe.js <file> <regex> [width] [limit]
const fs = require('fs');
const [,, file, reStr, widthStr, limitStr] = process.argv;
const width = widthStr ? +widthStr : 600;
const limit = limitStr ? +limitStr : 12;
const src = fs.readFileSync(file, 'utf8');
const re = new RegExp(reStr, 'g');
let m, n = 0;
while ((m = re.exec(src)) !== null && n < limit) {
  const s = Math.max(0, m.index - width);
  const e = Math.min(src.length, m.index + m[0].length + width);
  console.log(`\n===== hit ${++n} @ ${m.index} (len=${m[0].length}) =====`);
  console.log(src.slice(s, e).replace(/\n/g, '\\n'));
  if (re.lastIndex === m.index) re.lastIndex++;
}
console.log(`\n[${n}] matches shown for /${reStr}/`);
