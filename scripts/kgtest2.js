// Probe kugou gateway + get_res_privilege + lyric with correct field names.
const http = require('http');
const { URL } = require('url');
function call(method, urlStr, headers = {}, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const data = body ? JSON.stringify(body) : null;
    const h = { 'User-Agent': 'Mozilla/5.0', ...headers, ...(data ? { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) } : {}) };
    const r = http.request({ hostname: u.hostname, path: u.pathname + u.search, method, headers: h, timeout: 20000 }, res => {
      let d = ''; res.on('data', c => (d += c)); res.on('end', () => resolve({ status: res.statusCode, body: d }));
    });
    r.on('error', reject); r.on('timeout', () => r.destroy(new Error('timeout')));
    if (data) r.write(data);
    r.end();
  });
}
const KG = { 'KG-RC': '1', 'KG-THash': 'expand_search_manager.cpp:852736169:451', 'User-Agent': 'KuGou2012-9020-ExpandSearchManager' };
const GW = { 'KG-THash': '13a3164', 'KG-RC': '1', 'KG-Fake': '0', 'KG-RF': '00869891', 'x-router': 'kmr.service.kugou.com', 'User-Agent': 'Android712-AndroidPhone-11451-376-0-FeeCacheUpdate-wifi' };

(async () => {
  const kw = process.argv[2] || '晴天';
  const s = JSON.parse((await call('GET', `http://songsearch.kugou.com/song_search_v2?platform=AndroidFilter&iscorrection=1&keyword=${encodeURIComponent(kw)}&hifiquality=0&pagesize=20&PrivilegeFilter=0&page=1`)).body);
  const free = s.data.lists.find(x => x.Privilege !== 10 && x.FileHash) || s.data.lists.find(x => x.FileHash);
  const it = free || s.data.lists[0];
  console.log('pick:', it.OriSongName, '/', it.SingerName, '| audioid', it.Audioid, '| albumid', it.AlbumID, '| priv', it.Privilege);
  const hash = it.FileHash;

  console.log('\n=== gateway (ms clienttime) ===');
  let g = await call('POST', 'http://gateway.kugou.com/v2/album_audio/audio', GW, {
    area_code: '1', show_privilege: 1, show_album_info: '1', is_publish: '',
    appid: 1005, clientver: 11451, mid: '1', dfid: '-', clienttime: Date.now(),
    key: 'OIlwieks28dk2k092lksi2UIkp',
    fields: 'album_info,author_name,audio_info,ori_audio_name,base,songname',
    data: [{ hash }],
  });
  console.log(g.body.slice(0, 900));

  console.log('\n=== get_res_privilege ===');
  let p = await call('POST', 'http://media.store.kugou.com/v1/get_res_privilege', KG, {
    appid: 1001, area_code: '1', behavior: 'play', clientver: '9020',
    need_hash_offset: 1, relate: 1,
    resource: [{ album_audio_id: it.Audioid, album_id: it.AlbumID, hash, id: 0, name: `${it.SingerName} - ${it.OriSongName}.mp3`, type: 'audio' }],
    token: '', userid: 2626431536, vip: 1,
  });
  console.log(p.body.slice(0, 1500));

  console.log('\n=== lyric ===');
  let l = await call('GET', `http://lyrcs.kugou.com/search?ver=1&man=yes&client=pc&keyword=${encodeURIComponent(it.OriSongName)}&hash=${hash}&timelength=${it.Duration}&lrctxt=1`, KG);
  console.log('search:', l.body.slice(0, 300));
})().catch(e => { console.error('FATAL', e.message); process.exit(1); });
