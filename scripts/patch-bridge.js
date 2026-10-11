// Copy the cached bridge and add request logging so a user source's network
// traffic is visible. The patched file lives next to this script, never next to
// the downloaded source.
const fs = require("fs");
const src = process.argv[2];
const dst = process.argv[3];
let c = fs.readFileSync(src, "utf8");

c = c.replace(
  "function doRequest(method, target, opts) {",
  'function doRequest(method, target, opts) { process.stderr.write("REQ [" + (method || "") + "] " + target + "\\n");'
);
c = c.replace(
  "        resolve(out);",
  '        process.stderr.write("  <- " + res.statusCode + " " + buf.length + "B\\n"); ' +
    'process.stderr.write("  body: " + buf.toString("utf8").substring(0, 320).replace(/\\s+/g, " ") + "\\n"); ' +
    "resolve(out);"
);
c = c.replace(
  'if (body != null && body !== "") req.write(body);',
  "process.stderr.write(\"  reqbody: \" + String(body == null ? \"\" : body.toString(\"utf8\")).substring(0, 300).replace(/\\s+/g, \" \") + \"\\n\"); " +
    'if (body != null && body !== "") req.write(body);'
);
c = c.replace(
  "    req.on(\"error\", reject);",
  "    req.on(\"error\", function (e) { process.stderr.write('  ! ' + e.code + ' ' + e.message + \"\\n\"); reject(e); });"
);
// Surface the full stack so the failing line can be located.
c = c.replace(
  "String(e && e.message ? e.message : e)",
  "(e && e.stack ? e.stack : String(e))"
);
// Print the request headers so a signed request can be replayed by hand.
c = c.replace(
  "process.stderr.write(\"  reqbody: ",
  "process.stderr.write(\"  hdrs: \" + JSON.stringify(opts.headers || {}) + \"\\n\"); process.stderr.write(\"  reqbody: "
);
// Print the request handler the source registers, so its calling convention is
// visible without having to unobfuscate the whole file.
c = c.replace(
  "lx.on = function (name, fn) {",
  "lx.on = function (name, fn) { if (typeof fn === 'function') process.stderr.write('ON_HANDLER\\n' + fn.toString() + '\\n---END_HANDLER---\\n');"
);

fs.writeFileSync(dst, c);
console.log("patched: " + c.length + " bytes");
console.log("REQ markers: " + (c.match(/REQ \[/g) || []).length);
console.log("status markers: " + (c.match(/res\.statusCode \+ " /g) || []).length);
console.log("err markers: " + (c.match(/e\.code/g) || []).length);
