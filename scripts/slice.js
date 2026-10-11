// Pretty-print a character range of a bundle.
// Usage: node slice.js <bundle.js> <out.js> <startChar> <endChar>
// Also prints offsets of marker strings when given as extra args.
const fs = require('fs');
const [,, bundle, output, startStr, endStr, ...markers] = process.argv;
const src = fs.readFileSync(bundle, 'utf8');
const start = startStr ? +startStr : 0;
const end = endStr ? +endStr : src.length;

for (const mk of markers) {
  const i = src.indexOf(mk);
  console.log(`${mk.padEnd(34)} ${i >= 0 ? i : 'NOT FOUND'}`);
}

function fmt(s) {
  const out = [];
  let i = 0, d = 0, mode = null;
  const ind = () => '  '.repeat(Math.max(0, d));
  while (i < s.length) {
    const c = s[i], n = s[i + 1] || '';
    if (mode === '"' || mode === "'") {
      if (c === '\\') { out.push(c, n); i += 2; continue; }
      if (c === mode) { out.push(c); mode = null; i++; continue; }
      out.push(c); i++; continue;
    }
    if (mode === '`') {
      if (c === '\\') { out.push(c, n); i += 2; continue; }
      if (c === '`') { out.push(c); mode = null; i++; continue; }
      out.push(c); i++; continue;
    }
    if (mode === '//' || mode === '/*') {
      out.push(c); i++;
      if (mode === '/*' && c === '*' && n === '/') { out.push(n); i++; mode = null; continue; }
      if (c === '\n') mode = null;
      continue;
    }
    if (c === '"' || c === "'") { mode = c; out.push(c); i++; continue; }
    if (c === '`') { mode = '`'; out.push(c); i++; continue; }
    if (c === '/' && n === '/') { mode = '//'; out.push(c); i++; continue; }
    if (c === '/' && n === '*') { mode = '/*'; out.push(c); i++; continue; }
    if (c === '/') {
      let p = i - 1;
      while (p >= 0 && /\s/.test(s[p])) p--;
      const prev = p >= 0 ? s[p] : '';
      if ('=(,:[!&|?{};~+-*%<>^'.includes(prev)) {
        out.push(c); i++;
        while (i < s.length) {
          const r = s[i];
          out.push(r); i++;
          if (r === '\\') { out.push(s[i]); i++; continue; }
          if (r === '[') {
            while (i < s.length) {
              out.push(s[i]); i++;
              if (s[i - 1] === '\\') { out.push(s[i]); i++; continue; }
              if (s[i - 1] === ']') break;
            }
            continue;
          }
          if (r === '/') break;
        }
        continue;
      }
    }
    if (c === '{') { d++; out.push(' {\n' + ind()); i++; continue; }
    if (c === '}') { d = Math.max(0, d - 1); out.push('\n' + ind() + '}'); i++; continue; }
    if (c === ';') { out.push(';\n' + ind()); i++; continue; }
    if (c === ',') { out.push(',\n' + ind()); i++; continue; }
    if (c === '\n' || c === '\r') { out.push('\n' + ind()); i++; continue; }
    if (c === '=') { out.push(' = '); i++; continue; }
    if (c === ':') { out.push(': '); i++; continue; }
    out.push(c); i++;
  }
  return out.join('');
}

const body = src.slice(start, Math.min(end, src.length));
fs.writeFileSync(output, `// slice [${start},${Math.min(end, src.length)}) of ${bundle}\n${fmt(body)}`);
console.log(`wrote ${output}: ${fs.readFileSync(output, 'utf8').split('\n').length} lines`);
