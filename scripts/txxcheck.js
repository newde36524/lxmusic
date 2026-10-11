// Settle what X(e,t,n) really is for t>=32.
const { X } = require("C:/lx-rl/scripts/txjs.js");
const e = Buffer.from([33, 64, 35, 41, 40, 78, 72, 76]);
for (const t of [33, 37, 41, 44, 49, 53, 56, 63]) {
  const idx = 4 * (t / 32 | 0) + 3 - (t % 32 / 8 | 0);
  console.log(
    "t=" + t +
    " idx=" + idx +
    " byte=" + e[idx] +
    " bitpos=" + (7 - (t % 8)) +
    " bitval=" + ((e[idx] >>> (7 - (t % 8))) & 1) +
    " n=" + (31 - t) +
    " X=" + X(e, t, 31 - t).toString(16)
  );
}
