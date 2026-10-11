package lx

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Tencent's lyric payload is not base64 and not XOR: it is a bespoke 64-bit
// block cipher, applied 8 bytes at a time to the hex-encoded blob, then
// zlib-inflated. This file is a literal port of the routine in the desktop
// bundle (renderer.js, ParseLyricService). The bundle names are single
// letters, so they are kept here: X/Z/Q/J bit helpers, F the round function,
// R the 16-round Feistel body, ne the key schedule, oe the three passes.
//
// Every operation in the source is on 32-bit unsigned integers. Go's uint32
// shifts and `<<`/`>>` match JS's `>>>` and the low-32-bit truncation of `<<`,
// so the port needs no masking beyond what the source already does.

// txH: eight 64-entry nibble tables.
var txH = [8][64]uint8{
	{14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7, 0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8, 4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0, 15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13},
	{15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10, 3, 13, 4, 7, 15, 2, 8, 15, 12, 0, 1, 10, 6, 9, 11, 5, 0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15, 13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9},
	{10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8, 13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1, 13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7, 1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12},
	{7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15, 13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9, 10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4, 3, 15, 0, 6, 10, 10, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14},
	{2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9, 14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6, 4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14, 11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3},
	{12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11, 10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8, 9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6, 4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13},
	{4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1, 13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6, 1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2, 6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12},
	{13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7, 1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2, 7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8, 2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11},
}

var txK = [16]uint{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}

var txG = [28]uint{56, 48, 40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17, 9, 1, 58, 50, 42, 34, 26, 18, 10, 2, 59, 51, 43, 35}

var txW = [28]uint{62, 54, 46, 38, 30, 22, 14, 6, 61, 53, 45, 37, 29, 21, 13, 5, 60, 52, 44, 36, 28, 20, 12, 4, 27, 19, 11, 3}

var txY = [48]uint{13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9, 22, 18, 11, 3, 25, 7, 15, 6, 26, 19, 12, 1, 40, 51, 30, 36, 46, 54, 29, 39, 50, 44, 32, 47, 43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31}

// txKeyBytes is the 24-byte key material: three 8-byte groups, used in the
// order 16:24, 8:16, 0:8.
var txKeyBytes = [24]byte{33, 64, 35, 41, 40, 42, 36, 37, 49, 50, 51, 90, 88, 67, 33, 64, 33, 64, 35, 41, 40, 78, 72, 76}

// txX puts bit (31-bit(t)) of the t-th input bit into position bit(n).
// bit(t) means bit t of the 64-bit block, byte-major from the most significant
// byte: byte index 4*(t/32) + 3 - (t%32)/8, bit (7-t%8) within it.
func txX(e [8]byte, t, n uint) uint32 {
	idx := 4*(t/32) + 3 - (t%32)/8
	return uint32((e[idx]>>uint(7-t%8))&1) << n
}

// txZ puts bit (31-t) of e into position n.
func txZ(e uint32, t, n uint) uint32 {
	return ((e >> (31 - t)) & 1) << n
}

// txQ copies bit (31-t) of e to bit (31-n): the high bit lands at position n.
func txQ(e uint32, t, n uint) uint32 {
	return ((e << t) & 0x80000000) >> n
}

// txJ is the 5-bit reversal used to index the S-boxes.
func txJ(e uint8) uint {
	return uint(e&32) | (uint(e&31) >> 1) | (uint(e&1) << 4)
}

// txF is the round function of the 64-bit Feistel cipher.
func txF(x uint32, k [6]byte) uint32 {
	// The bundle writes these as decimal; the hex equivalents are 0xF0000000,
	// 0xF000000, 0xF00000 and 0xF0000 - note they are not evenly spaced.
	n := txQ(x, 31, 0) | (x&0xF0000000)>>1 |
		txQ(x, 4, 5) | txQ(x, 3, 6) | (x&0xF000000)>>3 |
		txQ(x, 8, 11) | txQ(x, 7, 12) | (x&0xF00000)>>5 |
		txQ(x, 12, 17) | txQ(x, 11, 18) | (x&0xF0000)>>7 | txQ(x, 16, 23)
	o := txQ(x, 15, 0) | (x&0x0000F000)<<15 |
		txQ(x, 20, 5) | txQ(x, 19, 6) | (x&0x00000F00)<<13 |
		txQ(x, 24, 11) | txQ(x, 23, 12) | (x&0x000000F0)<<11 |
		txQ(x, 28, 17) | txQ(x, 27, 18) | (x&0x0000000F)<<9 | txQ(x, 0, 23)
	i := [6]byte{
		byte(n >> 24 & 255), byte(n >> 16 & 255), byte(n >> 8 & 255),
		byte(o >> 24 & 255), byte(o >> 16 & 255), byte(o >> 8 & 255),
	}
	for e := 0; e < 6; e++ {
		i[e] ^= k[e]
	}
	s := uint32(txH[0][txJ(i[0]>>2)])<<28 |
		uint32(txH[1][txJ((i[0]&3)<<4|i[1]>>4)])<<24 |
		uint32(txH[2][txJ((i[1]&15)<<2|i[2]>>6)])<<20 |
		uint32(txH[3][txJ(i[2]&63)])<<16 |
		uint32(txH[4][txJ(i[3]>>2)])<<12 |
		uint32(txH[5][txJ((i[3]&3)<<4|i[4]>>4)])<<8 |
		uint32(txH[6][txJ((i[4]&15)<<2|i[5]>>6)])<<4 |
		uint32(txH[7][txJ(i[5]&63)])
	return txQ(s, 15, 0) | txQ(s, 6, 1) | txQ(s, 19, 2) | txQ(s, 20, 3) |
		txQ(s, 28, 4) | txQ(s, 11, 5) | txQ(s, 27, 6) | txQ(s, 16, 7) |
		txQ(s, 0, 8) | txQ(s, 14, 9) | txQ(s, 22, 10) | txQ(s, 25, 11) |
		txQ(s, 4, 12) | txQ(s, 17, 13) | txQ(s, 30, 14) | txQ(s, 9, 15) |
		txQ(s, 1, 16) | txQ(s, 7, 17) | txQ(s, 23, 18) | txQ(s, 13, 19) |
		txQ(s, 31, 20) | txQ(s, 26, 21) | txQ(s, 2, 22) | txQ(s, 8, 23) |
		txQ(s, 18, 24) | txQ(s, 12, 25) | txQ(s, 29, 26) | txQ(s, 5, 27) |
		txQ(s, 21, 28) | txQ(s, 10, 29) | txQ(s, 3, 30) | txQ(s, 24, 31)
}

// txSplit turns the 8 input bytes into the two 32-bit Feistel halves.
func txSplit(e [8]byte) (uint32, uint32) {
	var a, b uint32
	for g := 0; g < 4; g++ {
		for p := 0; p < 8; p++ {
			t := 57 + 2*g - 8*p
			dst := 31 - 8*g - p
			a |= txX(e, uint(t), uint(dst))
			b |= txX(e, uint(t-1), uint(dst))
		}
	}
	return a, b
}

// txPack writes the two halves back out as 8 bytes.
func txPack(hi, lo uint32) [8]byte {
	var n [8]byte
	n[3] = byte(txZ(lo, 7, 7) | txZ(hi, 7, 6) | txZ(lo, 15, 5) | txZ(hi, 15, 4) | txZ(lo, 23, 3) | txZ(hi, 23, 2) | txZ(lo, 31, 1) | txZ(hi, 31, 0))
	n[2] = byte(txZ(lo, 6, 7) | txZ(hi, 6, 6) | txZ(lo, 14, 5) | txZ(hi, 14, 4) | txZ(lo, 22, 3) | txZ(hi, 22, 2) | txZ(lo, 30, 1) | txZ(hi, 30, 0))
	n[1] = byte(txZ(lo, 5, 7) | txZ(hi, 5, 6) | txZ(lo, 13, 5) | txZ(hi, 13, 4) | txZ(lo, 21, 3) | txZ(hi, 21, 2) | txZ(lo, 29, 1) | txZ(hi, 29, 0))
	n[0] = byte(txZ(lo, 4, 7) | txZ(hi, 4, 6) | txZ(lo, 12, 5) | txZ(hi, 12, 4) | txZ(lo, 20, 3) | txZ(hi, 20, 2) | txZ(lo, 28, 1) | txZ(hi, 28, 0))
	n[7] = byte(txZ(lo, 3, 7) | txZ(hi, 3, 6) | txZ(lo, 11, 5) | txZ(hi, 11, 4) | txZ(lo, 19, 3) | txZ(hi, 19, 2) | txZ(lo, 27, 1) | txZ(hi, 27, 0))
	n[6] = byte(txZ(lo, 2, 7) | txZ(hi, 2, 6) | txZ(lo, 10, 5) | txZ(hi, 10, 4) | txZ(lo, 18, 3) | txZ(hi, 18, 2) | txZ(lo, 26, 1) | txZ(hi, 26, 0))
	n[5] = byte(txZ(lo, 1, 7) | txZ(hi, 1, 6) | txZ(lo, 9, 5) | txZ(hi, 9, 4) | txZ(lo, 17, 3) | txZ(hi, 17, 2) | txZ(lo, 25, 1) | txZ(hi, 25, 0))
	n[4] = byte(txZ(lo, 0, 7) | txZ(hi, 0, 6) | txZ(lo, 8, 5) | txZ(hi, 8, 4) | txZ(lo, 16, 3) | txZ(hi, 16, 2) | txZ(lo, 24, 1) | txZ(hi, 24, 0))
	return n
}

// txRound runs the 16-round Feistel on one 8-byte block with one subkey.
func txRound(e [8]byte, k [16][6]byte, out *[8]byte) {
	hi, lo := txSplit(e)
	for r := 0; r < 15; r++ {
		prev := lo
		lo = txF(lo, k[r]) ^ hi
		hi = prev
	}
	hi = txF(lo, k[15]) ^ hi
	*out = txPack(hi, lo)
}

// txSeed packs the 28 selected input bits of each half into the two 32-bit
// words the schedule starts from.
func txSeed(e [8]byte) (uint32, uint32) {
	var hi, lo uint32
	for t := 0; t < 28; t++ {
		hi |= txX(e, txG[t], uint(31-t))
		lo |= txX(e, txW[t], uint(31-t))
	}
	return hi, lo
}

// txKey expands an 8-byte key group into 16 subkeys. The bundle's second
// argument selects which subkey row round r lands in: 0 mirrors them
// (row 15-r) and 1 leaves them in order (row r).
func txKey(e [8]byte, mirror bool) [16][6]byte {
	var n [16][6]byte
	hi, lo := txSeed(e)
	for r := 0; r < 16; r++ {
		sh := txK[r]
		hi = (hi<<sh | hi>>(28-sh)) & 0xFFFFFFF0
		lo = (lo<<sh | lo>>(28-sh)) & 0xFFFFFFF0
		row := r
		if mirror {
			row = 15 - r
		}
		for e := 0; e < 24; e++ {
			n[row][e/8] |= byte(txZ(hi, txY[e], uint(7-e%8)))
		}
		for e := 24; e < 48; e++ {
			n[row][e/8] |= byte(txZ(lo, txY[e]-27, uint(7-e%8)))
		}
	}
	return n
}

// txSubkeys slices the 24-byte key the way the bundle does: groups 16:24,
// 8:16 and 0:8 become subkeys 0, 1 and 2, with the mirror flag taken from
// the bundle's ne(...) argument.
func txSubkeys() [3][16][6]byte {
	var g0, g1, g2 [8]byte
	copy(g0[:], txKeyBytes[16:24])
	copy(g1[:], txKeyBytes[8:16])
	copy(g2[:], txKeyBytes[0:8])
	return [3][16][6]byte{txKey(g0, true), txKey(g1, false), txKey(g2, true)}
}

// txPasses applies the three subkeys in turn, which is what one 8-byte block
// of ciphertext goes through.
func txPasses(in [8]byte, k [3][16][6]byte, out *[8]byte) {
	var a [8]byte
	txRound(in, k[0], &a)
	txRound(a, k[1], out)
	txRound(*out, k[2], &a)
	*out = a
}

// TencentDecryptLyric turns a PlayLyricInfo payload into the lyric text:
// hex-decode, run every 8-byte block through the cipher, then zlib-inflate.
// It answers "" for anything malformed, exactly like the bundle.
func TencentDecryptLyric(h string) string {
	b, err := hex.DecodeString(strings.TrimSpace(h))
	if err != nil || len(b) == 0 {
		return ""
	}
	return txInflate(txDecryptBlocks(b))
}

// txDecryptBlocks runs the cipher in place over every full 8-byte block.
// Any trailing partial block is left alone, as the bundle does.
func txDecryptBlocks(b []byte) []byte {
	k := txSubkeys()
	for i := 0; i+8 <= len(b); i += 8 {
		var blk, out [8]byte
		copy(blk[:], b[i:i+8])
		txPasses(blk, k, &out)
		copy(b[i:i+8], out[:])
	}
	return b
}

// txInflate handles the zlib stream. Tencent emits it with Z_SYNC_FLUSH, which
// truncates the tail right after the deflate end-of-stream block, so a strict
// full-stream read fails. Read until the error and keep what came out; if the
// wrapper turns out not to be zlib, retry as raw deflate without the header.
func txInflate(b []byte) string {
	var best string
	if r, err := zlib.NewReader(bytes.NewReader(b)); err == nil {
		best = txReadUntilError(r)
	}
	if len(b) >= 2 {
		if got := txReadUntilError(flate.NewReader(bytes.NewReader(b[2:]))); len(got) > len(best) {
			best = got
		}
	}
	return strings.TrimSpace(best)
}

func txReadUntilError(r io.ReadCloser) string {
	defer r.Close()
	var buf bytes.Buffer
	chunk := make([]byte, 64*1024)
	for {
		n, err := r.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
		}
		if err != nil {
			return buf.String()
		}
	}
}

// TencentDecryptLRC is TencentDecryptLyric plus the KRC to LRC reduction: QQ
// hands back the karaoke payload wrapped in a QrcInfos XML document, every
// line shaped "[start,duration]字(start,len)字(start,len)...". lx-music shows
// plain [mm:ss.xx] lines, so this strips the per-character timings.
func TencentDecryptLRC(h string) string {
	return txKRCtoLRC(TencentDecryptLyric(h))
}

var (
	txKRCGroup = regexp.MustCompile(`\(\s*\d+\s*,\s*\d*\s*\)`)
	txKRCLine  = regexp.MustCompile(`^\[(\d+),(\d*)\]`)
)

func txKRCtoLRC(qrc string) string {
	if !strings.Contains(qrc, "LyricContent=") {
		return qrc
	}
	body := qrc[strings.Index(qrc, "LyricContent=")+len("LyricContent="):]
	body = strings.TrimLeft(body, `"`)
	if i := strings.Index(body, `"/>`); i >= 0 {
		body = body[:i]
	} else if i := strings.LastIndex(body, `"`); i >= 0 {
		body = body[:i]
	}
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if m := txKRCLine.FindStringSubmatchIndex(line); m != nil {
			start := line[m[2]:m[3]]
			text := txKRCGroup.ReplaceAllString(line[m[1]:], "")
			// A grouped phrase like ((1440,160)Jay(1600,160)...) loses its
			// timings but keeps its wrapper parens; fold them back to one pair.
			text = strings.ReplaceAll(text, "((", "(")
			text = strings.ReplaceAll(text, "))", ")")
			out = append(out, "["+txMMSS(start)+"]"+strings.TrimSpace(text))
			continue
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// txMMSS formats milliseconds the way LRC writes them, "[mm:ss.xx]".
func txMMSS(ms string) string {
	n, err := strconv.Atoi(ms)
	if err != nil || n < 0 {
		n = 0
	}
	return fmt.Sprintf("%02d:%02d.%02d", n/60000, (n/1000)%60, (n/10)%100)
}
