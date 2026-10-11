// Sacrificial comment line (the transport drops a leading line sometimes).
// Probe the kugou KRC payload and locate the real decoder source.
const fs = require('fs');
const zlib = require('zlib');
const http = require('http');
const { URL } = require('url');

function req(urlStr, headers = {}) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const r = http.request({ hostname: u.hostname, path: u.pathname + u.search, method: 'GET', headers: { 'User-Agent': 'Mozilla/5.0', ...headers }, timeout: 20000 }, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks) }));
    });
    r.on('error', reject); r.on('timeout', () => r.destroy(new Error('timeout')));
    r.end();
  });
}
const KG = { 'KG-RC': '1', 'KG-THash': 'expand_search_manager.cpp:852736169:451', 'User-Agent': 'KuGou2012-9020-ExpandSearchManager' };

async function main() {
  const kw = process.argv[2] || 'hello';
  const s = JSON.parse((await req(`http://songsearch.kugou.com/song_search_v2?platform=AndroidFilter&iscorrection=1&keyword=${encodeURIComponent(kw)}&hifiquality=0&pagesize=30&PrivilegeFilter=0&page=1`)).body.toString());
  const it = s.data.lists.find(x => x.Privilege !== 10 && x.FileHash) || s.data.lists.find(x => x.FileHash);
  console.log('pick:', it.OriSongName, '/', it.SingerName, '| priv', it.Privilege);
  const hash = it.FileHash;

  const l = JSON.parse((await req(`http://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword=${encodeURIComponent(it.OriSongName)}&hash=${hash}&timelength=${it.Duration}&lrctxt=1`, KG)).body.toString());
  console.log('candidates:', l.candidates.length, JSON.stringify(l.candidates && l.candidates[0]));
  const c = l.candidates[0];
  if (!c) return console.log('no candidate');
  const fmt = c.krctype === 1 && c.contenttype !== 1 ? 'krc' : 'lrc';
  const dl = JSON.parse((await req(`http://lyrics.kugou.com/download?ver=1&client=pc&id=${c.id}&accesskey=${c.accesskey}&fmt=${fmt}&charset=utf8`, KG)).body.toString());
  console.log('download fmt:', dl.fmt, 'content len:', String(dl.content).length);
  const raw = Buffer.from(String(dl.content), 'base64');
  console.log('decoded bytes:', raw.length, 'hex head:', raw.slice(0, 48).toString('hex'));
  console.log('utf8 head:', JSON.stringify(raw.toString('utf8').slice(0, 120)));
  for (const f of [['gunzipSync', zlib.gunzipSync], ['inflateSync', zlib.inflateSync], ['inflateRawSync', zlib.inflateRawSync]]) {
    try {
      const o = f[1](raw);
      console.log('  OK', f[0], '->', o.length, o.toString('hex').slice(0, 48), JSON.stringify(o.toString('utf8').slice(0, 80)));
    } catch (e) { /* keep going */ }
  }
  fs.writeFileSync('C:/lx-rl/probe/krc.bin', raw);

  // XOR the ciphertext with the magic string and a few other candidates.
  for (const key of ['FLaw^2tGQ61-', 'krc', 'kuwo']) {
    const kb = Buffer.from(key, 'utf8');
    const out = Buffer.from(raw);
    for (let i = 0; i < out.length; i++) out[i] ^= kb[i % kb.length];
    console.log('xor', JSON.stringify(key), '->', out.slice(0, 24).toString('hex'), JSON.stringify(out.toString('utf8').slice(0, 60)));
  }

  // Where does the decoder live?
  const dir = 'C:/lx-rl/extracted/dist/';
  for (const f of ['renderer.js', 'renderer-lyric.js', 'renderer.download.worker.js', 'main.js', 'user-api-preload.js']) {
    const t = fs.readFileSync(dir + f, 'utf8');
    console.log(f, 'len', t.length, '| FLaw', t.indexOf('FLaw'), '| krc1', t.indexOf('krc1'), '| 0x60krc', t.indexOf('krc'));
  }
}
main().catch(e => { console.error('FATAL', e); process.exit(1); });
