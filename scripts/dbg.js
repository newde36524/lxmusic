// Debug: report every webpack module with its size and which keywords it contains.
const fs = require('fs');
const path = require('path');
const [,, bundle, outDir, ...keywords] = process.argv;
const src = fs.readFileSync(bundle, 'utf8');

function findEnd(openBrace) {
  let d = 0, mode = null;
  for (let j = openBrace; j < src.length; j++) {
    const c = src[j], n = src[j + 1] || '';
    if (mode === '"' || mode === "'") { if (c === '\\') { j++; continue; } if (c === mode) mode = null; continue; }
    if (mode === '`') { if (c === '\\') { j++; continue; } if (c === '`') mode = null; continue; }
    if (mode === '//' || mode === '/*') {
      if (mode === '/*' && c === '*' && n === '/') { mode = null; j++; }
      else if (mode === '//' && c === '\n') mode = null;
      continue;
    }
    if (c === '"' || c === "'") { mode = c; continue; }
    if (c === '`') { mode = '`'; continue; }
    if (c === '/' && n === '/') { mode = '//'; continue; }
    if (c === '/' && n === '*') { mode = '/*'; continue; }
    if (c === '/') {
      let p = j - 1;
      while (p >= 0 && /\s/.test(src[p])) p--;
      const prev = p >= 0 ? src[p] : '';
      if ('=(,:[!&|?{};~+-*%<>^'.includes(prev)) {
        j++;
        while (j < src.length) {
          if (src[j] === '\\') { j += 2; continue; }
          if (src[j] === '[') {
            j++;
            while (j < src.length) {
              if (src[j] === '\\') { j += 2; continue; }
              if (src[j] === ']') { j++; break; }
              j++;
            }
            continue;
          }
          j++;
          if (src[j - 1] === '/') break;
        }
        continue;
      }
    }
    if (c === '{') d++;
    else if (c === '}') { d--; if (d === 0) return j; }
  }
  return -1;
}

const idRe = /\d{4,6}\(e,t,n\)\{/g;
let m, prevEnd = -1;
const rows = [];
while ((m = idRe.exec(src)) !== null) {
  const id = m[0].slice(0, m[0].length - '(e,t,n){'.length);
  const openBrace = m.index + m[0].length - 1;
  const end = findEnd(openBrace);
  const body = end > 0 ? src.slice(m.index, end + 1) : '';
  rows.push({ id, at: m.index, len: body.length, end, hits: keywords.filter(k => body.includes(k)) });
  idRe.lastIndex = end > 0 ? end + 1 : m.index + m[0].length;
}
rows.sort((a, b) => b.len - a.len);
for (const r of rows.slice(0, 15)) {
  console.log(`${r.id.padEnd(7)} at=${String(r.at).padStart(8)} len=${String(r.len).padStart(8)} hits=[${r.hits.join('|')}]`);
}
console.log('total modules:', rows.length, 'with hits:', rows.filter(r => r.hits.length).length);

// find the module that should contain song_search_v2 and show where findEnd stopped
const probe = src.indexOf('song_search_v2');
const enc = rows.find(r => r.at <= probe && r.end >= probe);
console.log('probe at', probe, '=> enclosing module', enc && enc.id, 'at', enc && enc.at, 'end', enc && enc.end, 'len', enc && enc.len);
if (enc) {
  console.log('--- source around reported end of module ' + enc.id + ' ---');
  console.log(JSON.stringify(src.slice(enc.end - 250, enc.end + 250)));
}
// modules between 1297000 and 1310000
for (const r of rows.filter(r => r.at > 1297000 && r.at < 1310000).sort((a, b) => a.at - b.at)) {
  console.log(`${r.id.padEnd(7)} at=${String(r.at).padStart(8)} len=${String(r.len).padStart(8)} hits=[${r.hits.join('|')}]`);
}
