// Probe the kugou endpoints recovered from renderer.js to confirm response shapes.
const https = require('https');
const http = require('http');
const { URL } = require('url');

function req(method, urlStr, { headers = {}, body } = {}) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const lib = u.protocol === 'https:' ? https : http;
    const data = body ? (typeof body === 'string' ? body : JSON.stringify(body)) : null;
    const r = lib.request({
      hostname: u.hostname, port: u.port, path: u.pathname + u.search, method,
      headers: { ...headers, ...(data ? { 'Content-Length': Buffer.byteLength(data) } : {}) },
      timeout: 20000,
    }, res => {
      let d = '';
      res.on('data', c => (d += c));
      res.on('end', () => resolve({ status: res.statusCode, body: d, headers: res.headers }));
    });
    r.on('error', reject);
    r.on('timeout', () => r.destroy(new Error('timeout')));
    if (data) r.write(data);
    r.end();
  });
}

const KG_H = { 'KG-RC': '1', 'KG-THash': 'expand_search_manager.cpp:852736169:451', 'User-Agent': 'KuGou2012-9020-ExpandSearchManager' };
const GW_H = { 'KG-THash': '13a3164', 'KG-RC': '1', 'KG-Fake': '0', 'KG-RF': '00869891', 'x-router': 'kmr.service.kugou.com', 'User-Agent': 'Android712-AndroidPhone-11451-376-0-FeeCacheUpdate-wifi', 'Content-Type': 'application/json' };

(async () => {
  const kw = process.argv[2] || '晴天';
  const search = await req('GET', `http://songsearch.kugou.com/song_search_v2?platform=AndroidFilter&iscorrection=1&keyword=${encodeURIComponent(kw)}&hifiquality=0&pagesize=3&PrivilegeFilter=0&page=1`);
  const sj = JSON.parse(search.body);
  console.log('--- search status', search.status, 'error_code', sj.error_code, 'total', sj.data && sj.data.total);
  const it = sj.data.lists[0];
  console.log('audioid', it.audioid, 'albumid', it.albumid, 'albumname', it.albumname);
  console.log('name', it.ORISongName, it.suffix, '| singer', JSON.stringify((it.singers || []).map(s => s.name)));
  console.log('duration', it.duration, 'filesize', it.filesize, 'filehash', it.filehash);
  console.log('hq', it.HQFileSize, it.HQFileHash, '| sq', it.SQFileSize, it.SQFileHash, '| res', it.ResFileSize, it.ResFileHash);
  console.log('Grp', JSON.stringify(it.Grp));
  const hash = it.filehash, audioid = it.audioid;

  console.log('\n=== gateway album_audio/audio ===');
  const gw = await req('POST', 'http://gateway.kugou.com/v2/album_audio/audio', {
    headers: GW_H,
    body: {
      area_code: '1', show_privilege: 1, show_album_info: '1', is_publish: '',
      appid: 1005, clientver: 11451, mid: '1', dfid: '-', clienttime: Date.now(),
      key: 'OIlwieks28dk2k092lksi2UIkp',
      fields: 'album_info,author_name,audio_info,ori_audio_name,base,songname',
      data: [{ hash }],
    },
  });
  console.log('status', gw.status);
  console.log(JSON.stringify(JSON.parse(gw.body)).slice(0, 1800));

  console.log('\n=== media.store get_res_privilege ===');
  const pv = await req('POST', 'http://media.store.kugou.com/v1/get_res_privilege', {
    headers: { ...KG_H, 'Content-Type': 'application/json' },
    body: {
      appid: 1001, area_code: '1', behavior: 'play', clientver: '9020',
      need_hash_offset: 1, relate: 1,
      resource: [{ album_audio_id: audioid, album_id: it.albumid, hash, id: 0, name: `${(it.singers || []).map(s => s.name).join('、')} - ${it.ORISongName}.mp3`, type: 'audio' }],
      token: '', userid: 2626431536, vip: 1,
    },
  });
  const pj = JSON.parse(pv.body);
  console.log('status', pv.status, 'error_code', pj.error_code);
  console.log('info keys', pj.data && Object.keys(pj.data[0] && pj.data[0].info || {}));
  console.log(JSON.stringify(pj.data && pj.data[0]).slice(0, 1200));

  console.log('\n=== lyric search ===');
  const ly = await req('GET', `http://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword=${encodeURIComponent(it.ORISongName)}&hash=${hash}&timelength=${it.duration}&lrctxt=1`, { headers: KG_H });
  const lj = JSON.parse(ly.body);
  console.log('status', ly.status, 'candidates', JSON.stringify(lj.candidates && lj.candidates.slice(0, 2)));
  if (lj.candidates && lj.candidates[0]) {
    const c = lj.candidates[0];
    const fmt = c.krctype === 1 && c.contenttype !== 1 ? 'krc' : 'lrc';
    const dl = await req('GET', `http://lyrics.kugou.com/download?ver=1&client=pc&id=${c.id}&accesskey=${c.accesskey}&fmt=${fmt}&charset=utf8`, { headers: KG_H });
    const dj = JSON.parse(dl.body);
    console.log('dl status', dl.status, 'fmt', dj.fmt, 'content(base64?)', String(dj.content).slice(0, 40));
    try { console.log('decoded:', Buffer.from(dj.content, 'base64').toString('utf-8').slice(0, 120)); } catch (e) {}
  }
})().catch(e => { console.error('FATAL', e); process.exit(1); });
