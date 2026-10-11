// Spawn the bridge with a payload read straight from a UTF-8 file. Piping
// through PowerShell would let the console codepage mangle the Chinese song
// names in the payload, so keep the bytes inside node.
const { spawnSync } = require("child_process");
const fs = require("fs");

const bridge = process.argv[2];
const script = process.argv[3];
const payload = process.argv[4];

const r = spawnSync(process.execPath, [bridge, script], {
  input: fs.readFileSync(payload),
  timeout: 90 * 1000,
});
process.stdout.write(r.stdout ? r.stdout.toString("utf8") : "");
process.stderr.write(r.stderr ? r.stderr.toString("utf8") : "");
if (r.error) process.stderr.write(String(r.error));
process.exitCode = r.status || 0;
