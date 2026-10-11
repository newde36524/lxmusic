// Extract webpack module factories containing given keywords, pretty-printed.
// Usage: node mods2.js <bundle.js> <outDir> kw1 kw2 ...
const fs = require('fs');
const path = require('path');
const [,, bundle, outDir, ...keywords] = process.argv;
const src = fs.readFileSync(bundle, 'utf8');

// --- naive brace match (string/template/comment aware only) ---
function findEnd(openBrace) {
  let d = 0, mode = null;
  for (let j = openBrace; j < src.length; j++) {
    const c = src[j], n = src[j + 1] || '';
    if (mode === '"' || mode === "'") {
      if (c === '\\') { j++; continue; }
      if (c === mode) mode = null;
      continue;
    }
    if (mode === '`') {
      if (c === '\\') { j++; continue; }
      if (c === '`') mode = null;
      continue;
    }
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

// --- formatter (reuses modules.js logic) ---
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

fs.mkdirSync(outDir, { recursive: true });
const re = /,(e,t,n)\{/g;
// walk matches manually to also capture the numeric id before each factory
const idRe = /\d{4,6}\(e,t,n\)\{/g;
let m, found = 0, written = 0;
while ((m = idRe.exec(src)) !== null) {
  found++;
  const id = m[0].slice(0, m[0].length - '(e,t,n){'.length);
  const openBrace = m.index + m[0].length - 1;
  const end = findEnd(openBrace);
  if (end < 0) continue;
  const body = src.slice(m.index, end + 1);
  const hits = keywords.filter(k => body.includes(k));
  if (hits.length) {
    fs.writeFileSync(path.join(outDir, `mod_${id}.js`), `// ===== module ${id}  (${body.length} bytes, matches: ${hits.join(' | ')}) =====\n${fmt(body)}`);
    written++;
    console.log(`module ${id}: ${body.length} bytes  <- ${hits.join(' | ')}`);
  }
  idRe.lastIndex = end + 1; // skip past this module body
}
console.log(`scanned ${found} factories, wrote ${written}`);
