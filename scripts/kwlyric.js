// Sacrificial first line.
// Dump the kuwo lyric provider + its decompressor from renderer.js.
const fs = require('fs');
const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
const keys = ['wordTimeAll', 'wordLine'];
for (const key of keys) {
  let i = src.indexOf(key);
  let n = 0;
  while (i >= 0 && n < 4) {
    console.log('==== ' + key + ' at ' + i + ' ====');
    console.log(src.substring(Math.max(0, i - 1800), i + 1800).replace(/;/g, ';\n').replace(/,/g, ',\n'));
    console.log();
    i = src.indexOf(key, i + 1);
    n++;
  }
}
