// Dump the raw first search item from kugou song_search_v2.
const http = require('http');
const { URL } = require('url');
function get(urlStr) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    http.get({ hostname: u.hostname, path: u.pathname + u.search, headers: { 'User-Agent': 'Mozilla/5.0' }, timeout: 20000 }, res => {
      let d = ''; res.on('data', c => (d += c)); res.on('end', () => resolve(d));
    }).on('error', reject);
  });
}
get(`http://songsearch.kugou.com/song_search_v2?platform=AndroidFilter&iscorrection=1&keyword=${encodeURIComponent(process.argv[2] || '晴天')}&hifiquality=0&pagesize=1&PrivilegeFilter=0&page=1`)
  .then(d => {
    const j = JSON.parse(d);
    const it = j.data.lists[0];
    console.log('keys:', Object.keys(it).join(', '));
    console.log(JSON.stringify(it, null, 1));
  });
