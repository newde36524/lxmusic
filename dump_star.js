const fs=require('fs');
const want=process.argv[2];
const out=[];
for(const mf of fs.readdirSync('C:/lx-rl/extracted/dist').filter(f=>f.endsWith('.js.map'))){
  let map; try{map=JSON.parse(fs.readFileSync('C:/lx-rl/extracted/dist/'+mf,'utf8'));}catch(e){continue;}
  const names=map.sources.map((s,i)=>({s,i})).map(o=>({s:o.s.replace('webpack://lx-music-desktop/./','').replace(/\?[0-9a-f]+$/,''),i:o.i}));
  for(const o of names){
    if(o.s.toLowerCase().indexOf(want)>=0){
      const c=map.sourcesContent[o.i];
      out.push('######## '+mf+' :: '+o.s+' ('+(c?c.length:0)+' chars)\n'+(c||'NO CONTENT'));
    }
  }
}
fs.writeFileSync(process.argv[3],out.join('\n\n'));
console.log('sections:',out.length,'bytes:',out.join('').length);
