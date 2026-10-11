// Crude but string/regex/comment-aware JS pretty-printer for reading minified bundles.
// Usage: node pretty.js <in.js> <out.js>
const fs = require('fs');
const [,, input, output] = process.argv;
if (!input || !output) {
  console.error('usage: node pretty.js <in.js> <out.js>');
  process.exit(1);
}
const src = fs.readFileSync(input, 'utf8');
const out = [];
let i = 0, depth = 0, prevSig = '';
let mode = null; // null | '"' | "'" | '`' | '//' | '/*' | '/'
const NL = '\n';

function indent() { return '  '.repeat(Math.max(0, depth)); }

while (i < src.length) {
  const c = src[i];
  const n = src[i + 1] || '';

  // ---- literal state machines ----
  if (mode === '"' || mode === "'") {
    if (c === '\\') { out.push(c, n); i += 2; continue; }
    if (c === mode) { out.push(c); mode = null; i++; continue; }
    out.push(c); i++; continue;
  }
  if (mode === '`') {
    if (c === '\\') { out.push(c, n); i += 2; continue; }
    if (c === '`') { out.push(c); mode = null; i++; continue; }
    if (c === '$' && n === '{') { out.push('${'); i += 2; continue; }
    out.push(c); i++; continue;
  }
  if (mode === '//' || mode === '/*') {
    out.push(c); i++;
    if (mode === '/*' && c === '*' && n === '/') { out.push(n); i++; mode = null; continue; }
    if (c === '\n') mode = null;
    continue;
  }

  // ---- enter literal states ----
  if (c === '"' || c === "'") { mode = c; out.push(c); i++; continue; }
  if (c === '`') { mode = '`'; out.push(c); i++; continue; }
  if (c === '/' && n === '/') { mode = '//'; out.push(c); i++; continue; }
  if (c === '/' && n === '*') { mode = '/*'; out.push(c); i++; continue; }
  if (c === '/' && '=(,:[!&|?{};~'.includes(prevSig)) {
    // regex literal: copy until unescaped closing slash
    out.push(c); i++;
    let esc = false;
    while (i < src.length) {
      const r = src[i];
      out.push(r);
      if (r === '\\') { esc = true; i++; if (i < src.length) { out.push(src[i]); i++; } esc = false; continue; }
      if (r === '[') {
        i++;
        while (i < src.length) {
          out.push(src[i]);
          if (src[i] === '\\') { out.push(src[i + 1]); i += 2; continue; }
          if (src[i] === ']') { i++; break; }
          i++;
        }
        continue;
      }
      i++;
      if (r === '/') break;
    }
    prevSig = '/';
    continue;
  }

  // ---- structural ----
  if (c === '{') { depth++; out.push(c + NL); i++; prevSig = '{'; continue; }
  if (c === '}') { depth = Math.max(0, depth - 1); out.push(NL + indent() + '}'); i++; prevSig = '}'; continue; }
  if (c === ';') { out.push(';\n' + indent()); i++; prevSig = ';'; continue; }
  if (c === ',') { out.push(',\n' + indent()); i++; prevSig = ','; continue; }
  if (c === '\n' || c === '\r') { out.push(NL + indent()); i++; continue; }
  if (c === '=') { out.push(' = '); i++; prevSig = '='; continue; }
  if (c === ':') { out.push(': '); i++; prevSig = ':'; continue; }
  if (c === '(') { out.push('('); i++; prevSig = '('; continue; }
  if (c === ')') { out.push(')'); i++; prevSig = ')'; continue; }
  if (c === '[') { out.push('['); i++; prevSig = '['; continue; }
  if (c === ']') { out.push(']'); i++; prevSig = ']'; continue; }

  if (!/\s/.test(c)) prevSig = c;
  out.push(c); i++;
}
fs.writeFileSync(output, out.join(''));
const txt = fs.readFileSync(output, 'utf8');
console.log(`ok: ${txt.split('\n').length} lines -> ${output}`);
