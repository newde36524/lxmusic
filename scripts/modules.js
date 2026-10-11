// Split a webpack bundle into per-module pretty-printed files.
// Usage: node modules.js <bundle.js> <outDir> [keyword1 keyword2 ...]
const fs = require('fs');
const path = require('path');
const [,, bundle, outDir, ...keywords] = process.argv;

const src = fs.readFileSync(bundle, 'utf8');
const L = src.length;

// ---- pass 1: mark literal ranges and brace depth ----
const inLit = new Uint8Array(L);      // 1 = inside string/template/comment/regex
const depthAt = new Int32Array(L);    // brace depth before this char
let depth = 0, mode = null, i = 0;
while (i < L) {
  const c = src[i], n = src[i + 1] || '';
  depthAt[i] = depth;
  if (mode) { inLit[i] = 1; }
  if (mode === '"' || mode === "'") {
    if (c === '\\') { inLit[i + 1] = 1; i += 2; continue; }
    if (c === mode) mode = null;
    i++; continue;
  }
  if (mode === '`') {
    if (c === '\\') { inLit[i + 1] = 1; i += 2; continue; }
    if (c === '`') mode = null; else if (c === '$' && n === '{') mode = '${';
    i++; continue;
  }
  if (mode === '${') {
    if (c === '{') { depth++; i++; continue; }
    if (c === '}') { mode = '`'; i++; continue; }
    i++; continue;
  }
  if (mode === '//' || mode === '/*') {
    if (mode === '/*' && c === '*' && n === '/') { inLit[i + 1] = 1; mode = null; i += 2; continue; }
    if (mode === '//' && c === '\n') { mode = null; }
    i++; continue;
  }
  if (c === '"' || c === "'") { mode = c; i++; continue; }
  if (c === '`') { mode = '`'; i++; continue; }
  if (c === '/' && n === '/') { mode = '//'; i++; continue; }
  if (c === '/' && n === '*') { mode = '/*'; i++; continue; }
  if (c === '/') {
    let p = i - 1;
    while (p >= 0 && /\s/.test(src[p])) p--;
    const prev = p >= 0 ? src[p] : '';
    if ('=(,:[!&|?{};~+-*%<>^'.includes(prev)) {
      // regex literal
      let j = i;
      while (j < L) {
        inLit[j] = 1;
        if (src[j] === '\\') { inLit[j + 1] = 1; j += 2; continue; }
        if (src[j] === '[') {
          j++;
          while (j < L) {
            inLit[j] = 1;
            if (src[j] === '\\') { inLit[j + 1] = 1; j += 2; continue; }
            j++; if (src[j - 1] === ']') break;
          }
          continue;
        }
        j++;
        if (src[j - 1] === '/') break;
      }
      i = j; continue;
    }
  }
  if (c === '{') depth++;
  else if (c === '}') depth = Math.max(0, depth - 1);
  i++;
}

// ---- pass 2: locate module factories `12345(e,t,n){` ----
const starts = [];
for (let p = 0; (p = src.indexOf('(e,t,n){', p)) >= 0; ) {
  let q = p - 1;
  while (q >= 0 && src[q] >= '0' && src[q] <= '9') q--;
  const id = src.slice(q + 1, p);
  if (id.length >= 4 && !inLit[p]) starts.push({ id, bodyStart: p + '(e,t,n){'.length, start: q + 1 });
  p++;
}

// match closing brace
function matchBrace(openIdx) {
  let d = 0;
  for (let j = openIdx; j < L; j++) {
    if (inLit[j]) continue;
    if (src[j] === '{') d++;
    else if (src[j] === '}') { d--; if (d === 0) return j; }
  }
  return -1;
}

// ---- formatter (string/regex/comment aware) ----
function fmt(s) {
  const out = [];
  let i = 0, d = 0, prevSig = '', mode = null;
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
    if (!/\s/.test(c)) prevSig = c;
    out.push(c); i++;
  }
  return out.join('');
}

// ---- emit ----
fs.mkdirSync(outDir, { recursive: true });
const manifest = [];
for (const m of starts) {
  const end = matchBrace(m.bodyStart - 1);
  if (end < 0) continue;
  const body = src.slice(m.start, end + 1);
  const hits = keywords.filter(k => body.includes(k));
  manifest.push({ id: m.id, bytes: body.length, kw: hits.join(',') });
  if (keywords.length === 0 || hits.length) {
    fs.writeFileSync(path.join(outDir, `mod_${m.id}.js`), `// module ${m.id}\n${fmt(body)}`);
  }
}
manifest.sort((a, b) => b.bytes - a.bytes);
console.log(JSON.stringify(manifest.filter(m => m.kw).concat(keywords.length === 0 ? manifest.slice(0, 40) : []), null, 1));
