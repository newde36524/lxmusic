const fs=require('fs');
const src=fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js','utf8');
const i=src.indexOf('FLaw');
console.log('idx',i);
const w=src.substring(i-6000,i+9000);
const pretty=w.replace(/;/g,';\n').replace(/,/g,',\n');
fs.writeFileSync('C:/lx-rl/probe/krc.js',pretty);
console.log('window len',w.length,'pretty len',pretty.length);
