# Independent check of the ID3v2.4 / FLAC tags the CLI writes.
# Reads the file named on argv[1]; the writer is Go code, so this validates the
# bytes against the format specs rather than re-reading them with the same parser.
import struct
import sys

path = sys.argv[1]
data = open(path, "rb").read()
print(f"file: {path} ({len(data)} bytes)")

if data[:4] == b"fLaC":
    off, blocks = 4, []
    while off + 4 <= len(data):
        flag, size = data[off], int.from_bytes(data[off + 1:off + 4], "big")
        blocks.append((data[off] & 0x7F, data[off + 4:off + 4 + size]))
        off += 4 + size
        if flag & 0x80:
            break
    print("flac blocks:", [t for t, _ in blocks])
    names = {0: "streaminfo", 1: "padding", 2: "metadata", 3: "application", 4: "comment", 5: "seektable", 6: "picture"}
    for t, b in blocks:
        if t == 0:
            print(f"  streaminfo: {b[:4]!r} {len(b)} bytes")
        if t == 4:
            n = struct.unpack_from("<I", b, 0)[0]
            offc = 4
            for _ in range(n):
                ln = struct.unpack_from("<I", b, offc)[0]
                offc += 4
                k, _, v = b[offc:offc + ln].decode("utf-8").partition("=")
                offc += ln
                print(f"  {k} = {v[:60]!r}{'...' if len(v) > 60 else ''}")
        if t == 6:
            ptype = struct.unpack_from(">I", b, 0)[0]
            ml = struct.unpack_from(">I", b, 4)[0]
            mime = b[8:8 + ml].decode()
            dl = struct.unpack_from(">I", b, 8 + ml)[0]
            w, h, d, c = struct.unpack_from(">IIII", b, 8 + ml + dl + 4)
            datal = struct.unpack_from(">I", b, 8 + ml + dl + 4 + 16)[0]
            dstart = 8 + ml + dl + 4 + 16 + 4
            pic = b[dstart:dstart + datal]
            print(f"  picture: type={ptype} mime={mime} {w}x{h} depth={d} colors={c} {datal} bytes")
            print(f"  picture magic: {pic[:12]!r}")
    assert [t for t, _ in blocks][0] == 0, "streaminfo must stay first"
    print("audio bytes:", len(data) - off)
    sys.exit(0)

assert data[:3] == b"ID3", "not an ID3 file"
ver = data[3]
assert ver in (3, 4), f"version {ver}"
size = ((data[6] & 0x7F) << 21) | ((data[7] & 0x7F) << 14) | ((data[8] & 0x7F) << 7) | (data[9] & 0x7F)
# v2.3 counts frame sizes in synchsafe bytes too; v2.4 does not.
syn = ver == 3
print(f"id3v2.{ver} tag: {size} bytes of frames, audio starts at {10 + size}")
assert 10 + size <= len(data), "tag runs past the file"
assert data[10 + size:] and not data[10 + size:10 + size + 3] == b"ID3", "audio must follow the tag"


def fsize_at(off):
    raw = struct.unpack_from(">I", data, off + 4)[0]
    return ((raw >> 21 & 0x7F) << 21) | ((raw >> 14 & 0x7F) << 14) | ((raw >> 7 & 0x7F) << 7) | (raw & 0x7F) if syn else raw


off = 10
while off < 10 + size:
    fid = data[off:off + 4].decode("latin-1")
    fsize = fsize_at(off)
    flags = data[off + 8:off + 10]
    body = data[off + 10:off + 10 + fsize]
    assert flags == b"\x00\x00", f"{fid} has non-zero flags {flags!r}"
    assert off + 10 + fsize <= 10 + size, f"{fid} overruns the tag"
    print(f"{fid}  size={fsize}")
    if fid == "USLT":
        assert body[1:3] == b"\xff\xfe", "USLT encoding byte without a BOM"
        lang = body[3:6].decode()
        offd = 6
        while body[offd:offd + 2] != b"\x00\x00":
            offd += 2
        print(f"    lang={lang} lyric={body[offd + 2:].decode('utf-16-le')[:70]!r}")
    elif body[:1] == b"\x01":
        # UTF-16 with BOM, null-terminated
        assert body[1:3] == b"\xff\xfe", f"{fid} BOM {body[1:3]!r}"
        text = body[3:].rstrip(b"\x00").decode("utf-16-le")
        print(f"    text={text!r}")
    elif fid == "APIC":
        assert body[:1] == b"\x00", "APIC mime must be ISO-8859-1"
        offm = 1
        while body[offm] != 0:
            offm += 1
        mime = body[1:offm].decode("latin-1")
        assert body[offm] == 0
        ptype = body[offm + 1]
        offd = offm + 2
        assert body[offd] == 0, "APIC description must be empty here"
        pic = body[offd + 1:]
        print(f"    mime={mime} type={ptype} {len(pic)} bytes")
        print(f"    magic={pic[:12]!r}")
        assert mime == "image/" + pic[:4].lower().decode("latin-1", "replace") or mime.startswith("image/"), mime
    off += 10 + fsize
print(f"audio bytes after tag: {len(data) - (10 + size)}")
print("tag structure OK")
