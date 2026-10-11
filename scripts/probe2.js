// Probe kuwo lyric transport + dump the kugou KRC decoder module (92617).
const http = require('http');
const fs = require('fs');
const zlib = require('zlib');
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

function tryInflate(b, label) {
  try { console.log(`  [${label}] gunzipSync ->`, zlib.gunzipSync(b).toString('utf8').slice(0, 220).replace(/\n/g, '\\n')); return; } catch (e) {}
  try { console.log(`  [${label}] inflateSync ->`, zlib.inflateSync(b).toString('utf8').slice(0, 220).replace(/\n/g, '\\n')); return; } catch (e) {}
  try { console.log(`  [${label}] inflateRawSync ->`, zlib.inflateRawSync(b).toString('utf8').slice(0, 220).replace(/\n/g, '\\n')); return; } catch (e) {}
  console.log(`  [${label}] not compressed; head =`, b.toString('utf8').slice(0, 220).replace(/\n/g, '\\n'));
}

(async () => {
  console.log('=== kuwo lyric lrcx=1 ===');
  for (const lrcx of ['1', '0']) {
    const u = `http://mlyric.kuwo.cn/mobi.s?f=web&type=lyric&lrcx=${lrcx}&rid=228908&encode=utf8`;
    try {
      const r = await req(u);
      console.log(`lrcx=${lrcx} status=${r.status} ct=${r.headers['content-type']} enc=${r.headers['content-encoding']} len=${r.body.length}`);
      console.log('  body head:', r.body.toString('utf8').slice(0, 160).replace(/\n/g, '\\n'));
      tryInflate(r.body, `lrcx=${lrcx} raw`);
    } catch (e) { console.log('ERR', e.message); }
  }

  console.log('\n=== kuwo lyric lrcx=1 for a rid without krc ===');
  try {
    const r = await req(`http://mlyric.kuwo.cn/mobi.s?f=web&type=lyric&lrcx=1&rid=6239986&encode=utf8`);
    console.log('status', r.status, 'len', r.body.length, 'head:', r.body.toString('utf8').slice(0, 200).replace(/\n/g, '\\n'));
  } catch (e) { console.log('ERR', e.message); }

  console.log('\n=== renderer.js module 92617 (KRC decoder) ===');
  const src = fs.readFileSync('C:/lx-rl/extracted/dist/renderer.js', 'utf8');
  const pats = ['FLaw^2tGQ61-', '92617:'];
  for (const p of pats) {
    const i = src.indexOf(p);
    console.log(`-- "${p}" at ${i}`);
    if (i >= 0) fs.writeFileSync('C:/lx-rl/probe/mod92617.js', src.substring(Math.max(0, i - 4000), i + 6000));
  }
  console.log('written C:/lx-rl/probe/mod92617.js');
})().catch(e => { console.error('FATAL', e); process.exit(1); });
