// Sacrificial first line.
// Fetch a kuwo lyric and report the exact header bytes plus stream magic.
const http = require('http');
const https = require('https');
const zlib = require('zlib');

const rid = process.argv[2] || '80196894';
const url = `http://mlyric.kuwo.cn/mobi.s?f=web&type=lyric&lrcx=1&rid=${rid}&encode=utf8`;

function get(u) {
  return new Promise((res, rej) => {
    (u.startsWith('https') ? https : http).get(u, { headers: { 'User-Agent': 'kwplayer_ar_9.2.2.1', 'Referer': 'http://www.kuwo.cn/' } }, r => {
      const c = [];
      r.on('data', d => c.push(d));
      r.on('end', () => res({ status: r.statusCode, enc: r.headers['content-encoding'], body: Buffer.concat(c) }));
      r.on('error', rej);
    }).on('error', rej);
  });
}

get(url).then(({ status, enc, body }) => {
  console.log('status', status, 'encoding', enc, 'len', body.length);
  console.log('first40 hex', body.subarray(0, 40).toString('hex'));
  console.log('first40 json', JSON.stringify(body.subarray(0, 40).toString('latin1')));
  for (const marker of ['\n\n', '\r\n\r\n', '\r\n', '\n']) {
    const i = body.indexOf(marker);
    console.log('marker', JSON.stringify(marker), 'at', i);
  }
  // Try to locate a zlib/gzip/deflate start.
  let found = -1;
  for (let i = 0; i < body.length - 1; i++) {
    if (body[i] === 0x78 && (body[i + 1] === 0x9c || body[i + 1] === 0x01 || body[i + 1] === 0x5e || body[i + 1] === 0xda)) {
      found = i; break;
    }
  }
  console.log('zlib 0x78 at', found);
  let gz = -1;
  for (let i = 0; i < body.length - 1; i++) {
    if (body[i] === 0x1f && body[i + 1] === 0x8b) { gz = i; break; }
  }
  console.log('gzip 0x1f8b at', gz);
  // Try every plausible start and decompress.
  for (const start of [found, gz, 0]) {
    if (start < 0) continue;
    for (const mode of ['zlib', 'gunzip', 'inflateRaw']) {
      try {
        const out = zlib[mode](body.subarray(start)).toString();
        console.log('OK mode', mode, 'start', start, 'len', out.length);
        console.log(out.substring(0, 400));
        return;
      } catch (e) { /* next */ }
    }
  }
  console.log('nothing decompressed');
});
