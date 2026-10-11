// Probe candidate kugou playback-URL endpoints + the real lyric host.
const http = require('http');
const { URL } = require('url');
function req(method, urlStr, headers = {}, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const data = body ? JSON.stringify(body) : null;
    const h = { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36', ...headers, ...(data ? { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) } : {}) };
    const r = http.request({ hostname: u.hostname, path: u.pathname + u.search, method, headers: h, timeout: 20000 }, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks) }));
    });
    r.on('error', reject); r.on('timeout', () => r.destroy(new Error('timeout')));
    if (data) r.write(data);
    r.end();
  });
}
const KG = { 'KG-RC': '1', 'KG-THash': 'expand_search_manager.cpp:852736169:451', 'User-Agent': 'KuGou2012-9020-ExpandSearchManager' };

(async () => {
  const kw = process.argv[2] || '晴天';
  const s = JSON.parse((await req('GET', `http://songsearch.kugou.com/song_search_v2?platform=AndroidFilter&iscorrection=1&keyword=${encodeURIComponent(kw)}&hifiquality=0&pagesize=30&PrivilegeFilter=0&page=1`)).body.toString());
  const it = (s.data.lists.find(x => x.Privilege !== 10 && x.FileHash) || s.data.lists.find(x => x.FileHash));
  console.log('pick:', it.OriSongName, '/', it.SingerName, '| audioid', it.Audioid, '| albumid', it.AlbumID, '| priv', it.Privilege);
  const hash = it.FileHash, h320 = it.HQFileHash, hflac = it.SQFileHash;

  console.log('\n=== media.kugou.com/trackinfo/{hash} ===');
  for (const h of [hash, h320]) {
    if (!h) continue;
    try {
      const r = await req('GET', `http://media.kugou.com/trackinfo/${h}`);
      console.log(h.slice(0, 8), '->', r.status, 'location:', r.headers.location, '| bytes', r.body.length, r.body.slice(0, 4));
    } catch (e) { console.log(h.slice(0, 8), 'ERR', e.message); }
  }

  console.log('\n=== ll/service cache.get ===');
  try {
    const r = await req('GET', `http://www.kugou.com/ll/service/index.php?method=cache.get&album_audio_id=${it.Audioid}&hash=${hash}&hash_320=${h320}&hash_flac=${hflac}&plat=1&expire=10&v=1`, KG);
    console.log(r.status, r.body.toString().slice(0, 400));
  } catch (e) { console.log('ERR', e.message); }

  console.log('\n=== media.kugou.com/20180816/08/04/... (direct guess via get_res_privilege cd_url) ===');
  const pv = await req('POST', 'http://media.store.kugou.com/v1/get_res_privilege', { ...KG, 'Content-Type': 'application/json' }, {
    appid: 1001, area_code: '1', behavior: 'play', clientver: '9020', need_hash_offset: 1, relate: 1,
    resource: [{ album_audio_id: it.Audioid, album_id: it.AlbumID, hash, id: 0, name: `${it.SingerName} - ${it.OriSongName}.mp3`, type: 'audio' }],
    token: '', userid: 2626431536, vip: 1,
  });
  const pj = JSON.parse(pv.body.toString());
  console.log('privilege:', JSON.stringify(pj.data && pj.data[0] && pj.data[0].info).slice(0, 200));

  console.log('\n=== lyric (lyrics.kugou.com) ===');
  try {
    const l = await req('GET', `http://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword=${encodeURIComponent(it.OriSongName)}&hash=${hash}&timelength=${it.Duration}&lrctxt=1`, KG);
    const lj = JSON.parse(l.body.toString());
    console.log('search:', l.status, JSON.stringify(lj.candidates && lj.candidates.slice(0, 3)));
    if (lj.candidates && lj.candidates[0]) {
      const c = lj.candidates[0];
      const fmt = c.krctype === 1 && c.contenttype !== 1 ? 'krc' : 'lrc';
      const dl = await req('GET', `http://lyrics.kugou.com/download?ver=1&client=pc&id=${c.id}&accesskey=${c.accesskey}&fmt=${fmt}&charset=utf8`, KG);
      const dj = JSON.parse(dl.body.toString());
      console.log('download:', dl.status, 'fmt', dj.fmt, '| content head:', String(dj.content).slice(0, 60));
      try { console.log('decoded:', Buffer.from(dj.content, 'base64').toString('utf-8').slice(0, 200)); } catch (e) {}
    }
  } catch (e) { console.log('ERR', e.message); }
})().catch(e => { console.error('FATAL', e); process.exit(1); });
