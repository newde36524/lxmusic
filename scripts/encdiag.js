const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
const a = [102, 76, 97, 119];
console.log(src.indexOf(String.fromCharCode.apply(null, a)));
console.log(src.indexOf('zzc'));
