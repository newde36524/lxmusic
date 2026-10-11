// Verify the Tencent musics.fcg sign algorithm recovered from renderer.js
const crypto = require('crypto');
const _e = [23, 14, 6, 36, 16, 40, 7, 19];
const me = [16, 1, 32, 12, 19, 27, 8, 5];
const he = [89, 39, 179, 150, 218, 82, 58, 252, 177, 52, 186, 123, 120, 64, 242, 133, 143, 161, 121, 179];
const ge = (e, t) => t.map(i => e[i]).join('');

function fe(e) {
  const t = crypto.createHash('sha1').update(e).digest('hex');
  const n = ge(t, _e);
  const o = ge(t, me);
  const i = he.map((v, k) => v ^ parseInt(t.slice(2 * k, 2 * k + 2), 16));
  return `zzc${n}${Buffer.from(i).toString('base64').replace(/[\/+=]/g, '').replace(/[\/+=]/g, '')}${o}`.toLowerCase();
}

const payload = {
  comm: {
    _channelid: '0', _os_version: '6.2.9200-2', ct: '19', cv: '2151',
    guid: '1F70E520B2EAA7D25E11760783C53CA9', patch: '118',
    psrf_access_token_expiresAt: 0, psrf_qqaccess_token: '', psrf_qqopenid: '', psrf_qqunionid: '',
    tmeAppID: 'qqmusic', tmeLoginType: 0, uin: '0', wid: '7223299733393904640',
  },
  'music.search.SearchCgiService': {
    module: 'music.search.SearchCgiService',
    method: 'DoSearchForQQMusicDesktop',
    param: { grp: 1, num_per_page: 3, page_num: 1, query: process.argv[2] || '周杰伦', remoteplace: 'txt.newclient.top', search_type: 0, searchid: '94857209004935904455977345678901' },
  },
};
const body = JSON.stringify(payload);
const sign = fe(body);
console.log('sha1len', crypto.createHash('sha1').update(body).digest('hex').length);
console.log('sign =', sign);

const https = require('https');
const req = https.request({
  hostname: 'u.y.qq.com', path: '/cgi-bin/musics.fcg?sign=' + sign, method: 'POST',
  headers: { 'User-Agent': 'QQMusic 14090508(android 12)', 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(body) },
}, res => {
  let d = '';
  res.on('data', c => d += c);
  res.on('end', () => {
    console.log('status', res.statusCode);
    try {
      const j = JSON.parse(d);
      const svc = j['music.search.SearchCgiService'];
      console.log('code', j.code, 'svc.code', svc && svc.code, 'subcode', svc && svc.subcode);
      const list = svc && svc.data && svc.data.body && svc.data.body.items;
      if (list) {
        console.log('songs:', list.length);
        const f = list[0] && list[0].action && list[0].action.type && list[0].action.type.content && list[0].action.type.content.first_song;
        console.log(JSON.stringify(f ? { title: f.title, mid: f.mid, id: f.id, album_mid: f.album && f.album.mid, file: f.file, interval: f.interval, singer: (f.singer || []).map(s => s.name) } : null, null, 1).slice(0, 1200));
      } else {
        console.log(d.slice(0, 600));
      }
    } catch (e) { console.log('non-json:', d.slice(0, 600)); }
  });
});
req.on('error', e => console.log('ERR', e.message));
req.write(body);
req.end();
