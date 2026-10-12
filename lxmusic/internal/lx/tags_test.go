package lx

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// decodeID3 parses a v2.4 tag into frame payloads, checking the frame table as
// it goes. It lets a test assert on what was written rather than on bytes, and
// it fails loudly on a frame that does not end where its own size says it
// should.
func decodeID3(t *testing.T, b []byte) (int, map[string][]byte) {
	t.Helper()
	if len(b) < 10 || !bytes.Equal(b[:3], []byte("ID3")) {
		t.Fatalf("not an ID3 header: % x", b[:minInt(len(b), 16)])
	}
	if b[3] != 0x04 || b[4] != 0x00 {
		t.Fatalf("version = %d.%d, want 2.4", b[3], b[4])
	}
	n := int(b[6])<<21 | int(b[7])<<14 | int(b[8])<<7 | int(b[9])
	if n+10 > len(b) {
		t.Fatalf("tag claims %d bytes of frames but the buffer holds %d", n, len(b)-10)
	}
	frames := map[string][]byte{}
	for off := 10; off < 10+n; {
		if 10+sizeAt(b, off) > 10+n {
			t.Fatalf("frame %q claims %d bytes past the tag", b[off:off+4], sizeAt(b, off))
		}
		id := string(b[off : off+4])
		sz := sizeAt(b, off)
		frames[id] = b[off+10 : off+10+sz]
		off += 10 + sz
	}
	return n, frames
}

func sizeAt(b []byte, off int) int {
	return int(b[off+4])<<24 | int(b[off+5])<<16 | int(b[off+6])<<8 | int(b[off+7])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decodeUTF16LE reads a null-terminated UTF-16LE string, the way a player does:
// the terminating pair is not part of the text.
func decodeUTF16LE(b []byte) string {
	for len(b) >= 2 && b[len(b)-2] == 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-2]
	}
	out := []rune{}
	for i := 0; i+1 < len(b); i += 2 {
		out = append(out, rune(binary.LittleEndian.Uint16(b[i:])))
	}
	return string(out)
}

// id3Header builds a 10 byte ID3v2 header: "ID3", version, flags, then the
// size in the width that version uses.
func id3Header(ver, flags byte, size []byte) []byte {
	return append([]byte{'I', 'D', '3', ver, 0x00, flags}, size...)
}

func mustWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSynchsafe(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{1, []byte{0, 0, 0, 1}},
		{127, []byte{0, 0, 0, 127}},
		{128, []byte{0, 0, 1, 0}},
		{0x0fffffff, []byte{0x7f, 0x7f, 0x7f, 0x7f}}, // 28 bits all set
	} {
		if got := synchsafe(tc.n); !bytes.Equal(got, tc.want) {
			t.Errorf("synchsafe(%d) = % x, want % x", tc.n, got, tc.want)
		}
	}
	// The field holds 28 bits, so nothing at or above 2^28 can round-trip.
	for _, n := range []int{0, 1, 28, 0x0fffffff} {
		b := synchsafe(n)
		if got := int(b[0])<<21 | int(b[1])<<14 | int(b[2])<<7 | int(b[3]); got != n {
			t.Errorf("synchsafe round-trip %d = %d", n, got)
		}
	}
}

func TestID3TextFrames(t *testing.T) {
	_, frames := decodeID3(t, id3Tag(&Metadata{
		Title: "晴天", Artist: "周杰伦", Album: "叶惠美",
		Track: "3/12", Year: "2003", Genre: "流行",
	}))
	for id, want := range map[string]string{
		"TIT2": "晴天", "TPE1": "周杰伦", "TALB": "叶惠美",
		"TRCK": "3/12", "TDRC": "2003", "TCON": "流行",
	} {
		d, ok := frames[id]
		if !ok {
			t.Errorf("%s frame is missing", id)
			continue
		}
		if d[0] != 0x01 || d[1] != 0xff || d[2] != 0xfe {
			t.Errorf("%s is not UTF-16 with a BOM: % x", id, d[:3])
		}
		if got := decodeUTF16LE(d[3:]); got != want {
			t.Errorf("%s = %q, want %q", id, got, want)
		}
		if len(d) < 2 || d[len(d)-2] != 0 || d[len(d)-1] != 0 {
			t.Errorf("%s has no null terminator: % x", id, d)
		}
	}
}

// TestID3USLT pins the layout after the encoding byte: the three-byte language
// code, the description, then the lyric. A wrong offset reads as garbage in the
// player, so the positions are asserted directly.
func TestID3USLT(t *testing.T) {
	lyric := "[00:01.000]晴 天\n[00:03.000]周杰伦的歌"
	_, frames := decodeID3(t, id3Tag(&Metadata{Title: "晴天", Lyric: lyric}))
	d, ok := frames["USLT"]
	if !ok {
		t.Fatal("USLT frame is missing")
	}
	if string(d[3:6]) != "zho" {
		t.Errorf("language = %q, want zho", string(d[3:6]))
	}
	if d[6] != 0 || d[7] != 0 {
		t.Errorf("content description is not empty: % x", d[6:8])
	}
	if got := decodeUTF16LE(d[8:]); got != lyric {
		t.Errorf("lyric = %q, want %q", got, lyric)
	}
}

// TestID3APIC checks the cover frame and that the mime is sniffed from the bytes
// when the caller did not name one.
func TestID3APIC(t *testing.T) {
	pic := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	_, frames := decodeID3(t, id3Tag(&Metadata{Title: "晴天", Picture: pic}))
	d, ok := frames["APIC"]
	if !ok {
		t.Fatal("APIC frame is missing")
	}
	if d[0] != 0x00 {
		t.Errorf("encoding = %d, want ISO-8859-1", d[0])
	}
	mime := ""
	for _, c := range d[1:] {
		if c == 0 {
			break
		}
		mime += string(c)
	}
	if mime != "image/jpeg" {
		t.Errorf("mime = %q, want image/jpeg", mime)
	}
	if d[12] != 0x03 {
		t.Errorf("picture type = %d, want 3 (front cover)", d[12])
	}
	if d[13] != 0x00 {
		t.Errorf("description is not empty: % x", d[13])
	}
	if !bytes.Equal(d[14:], pic) {
		t.Error("the picture bytes were not preserved")
	}
}

// TestID3HeaderSize pins the invariant that the frame table exactly fills the
// space the header announces.
func TestID3HeaderSize(t *testing.T) {
	for _, m := range []*Metadata{
		{Title: "a"},
		{Title: "晴天", Artist: "周杰伦", Album: "叶惠美", Track: "3/12"},
		{Title: "a", Lyric: "[00:01.000]晴 天", Picture: []byte{0xff, 0xd8, 0xff, 0xe0, 1, 2}},
	} {
		tag := id3Tag(m)
		n, frames := decodeID3(t, tag)
		if n != len(tag)-10 {
			t.Errorf("tag claims %d frames, buffer holds %d", n, len(tag)-10)
		}
		if len(frames) == 0 {
			t.Error("no frames were written")
		}
	}
}

// TestID3Noop pins the empty case: a Metadata with nothing in it writes nothing,
// which is what keeps a nameless download from gaining an empty tag.
func TestID3Noop(t *testing.T) {
	if got := id3Tag(&Metadata{}); got != nil {
		t.Errorf("an empty Metadata produced a %d byte tag", len(got))
	}
	if got := id3Tag(nil); got != nil {
		t.Error("id3Tag(nil) produced a tag")
	}
}

func TestWriteMP3TagsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "晴天.mp3")
	audio := append([]byte{0xff, 0xfb, 0x90, 0x00}, bytes.Repeat([]byte{0xab}, 64)...)
	mustWrite(t, path, audio)
	if err := WriteTags(path, &Metadata{
		Title: "晴天", Artist: "周杰伦", Album: "叶惠美",
		Lyric: "[00:01.000]晴 天", Picture: []byte{0xff, 0xd8, 0xff, 0xe0, 1, 2, 3},
	}); err != nil {
		t.Fatalf("WriteTags: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, frames := decodeID3(t, got)
	for _, id := range []string{"TIT2", "TPE1", "TALB", "USLT", "APIC"} {
		if _, ok := frames[id]; !ok {
			t.Errorf("%s is missing from the saved file", id)
		}
	}
	if !bytes.HasSuffix(got, audio) {
		t.Error("the audio data was not preserved after the tag")
	}

	// A second write replaces the first tag instead of stacking on it.
	if err := WriteTags(path, &Metadata{Title: "新标题", Artist: "林俊杰"}); err != nil {
		t.Fatalf("re-tag: %v", err)
	}
	got, _ = os.ReadFile(path)
	if i := bytes.Index(got[10:], []byte("ID3")); i >= 0 {
		t.Error("the file carries two ID3 tags")
	}
	_, frames = decodeID3(t, got)
	if got := decodeUTF16LE(frames["TIT2"][3:]); got != "新标题" {
		t.Errorf("the re-tagged title = %q, want 新标题", got)
	}
	if _, ok := frames["USLT"]; ok {
		t.Error("a stale USLT frame survived the re-tag")
	}
	if !bytes.HasSuffix(got, audio) {
		t.Error("the re-tag lost the audio")
	}
}

// TestWriteMP3TagsStripsExisting pins the old tag's removal, which is what stops
// a player from showing the title the file had before.
func TestWriteMP3TagsStripsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.mp3")
	old := []byte{'I', 'D', '3', 0x03, 0x00, 0x00, 0x00, 0x00, 0x02, 0x30} // v2.3, 304 bytes
	old = append(old, bytes.Repeat([]byte{0x00}, 560)...)
	old = append(old, 0xff, 0xfb, 0x90, 0x00)
	mustWrite(t, path, old)
	if err := WriteTags(path, &Metadata{Title: "新"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.HasPrefix(got, []byte("ID3")) || got[3] != 0x04 {
		t.Errorf("the file does not start with the new v2.4 tag: % x", got[:4])
	}
	if i := bytes.Index(got[10:], []byte("ID3")); i >= 0 {
		t.Error("the old tag was left behind")
	}
}

func TestWriteMP3TagsRejectsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.mp3")
	mustWrite(t, path, nil)
	if err := WriteTags(path, &Metadata{Title: "t"}); err == nil {
		t.Error("an empty file was tagged as if it were audio")
	}
}

func TestStripID3v2(t *testing.T) {
	audio := []byte{0xff, 0xfb}
	pad := 256
	zeros := bytes.Repeat([]byte{0}, pad)

	// The size is derived from the payload with the same encoder, so a fixture
	// can never claim a length its own padding does not have.
	v3 := append(id3Header(3, 0x00, synchsafe(pad)), zeros...)
	v3 = append(v3, audio...)
	if got := stripID3v2(v3); !bytes.Equal(got, audio) {
		t.Errorf("v2.3 tag was not stripped: %d bytes left", len(got))
	}

	// v2.2 counts the size in plain bytes, not synchsafe.
	v2 := append(id3Header(2, 0x00, []byte{0x00, byte(pad >> 8), byte(pad)}), zeros...)
	v2 = append(v2, audio...)
	if got := stripID3v2(v2); !bytes.Equal(got, audio) {
		t.Error("v2.2 tag was not stripped")
	}

	foot := append(id3Header(4, 0x10, synchsafe(pad)), zeros...)
	foot = append(foot, '3', 'D', 'I', 0x04, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00) // footer
	foot = append(foot, audio...)
	if got := stripID3v2(foot); !bytes.Equal(got, audio) {
		t.Error("the footer was not stripped")
	}

	if got := stripID3v2(audio); !bytes.Equal(got, audio) {
		t.Error("plain audio was stripped")
	}
	if got := stripID3v2([]byte("ID3garbage")); !bytes.Equal(got, []byte("ID3garbage")) {
		t.Error("an unrecognised ID3 header was mangled")
	}
	// A size that runs past the end of the file is not a tag to remove.
	short := []byte{'I', 'D', '3', 0x04, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00}
	short = append(short, 0x00, 0x00)
	if got := stripID3v2(short); !bytes.Equal(got, short) {
		t.Error("a truncated tag was stripped anyway")
	}
}

func TestWriteTagsSkipsUnknownFormats(t *testing.T) {
	dir := t.TempDir()
	for _, ext := range []string{".ape", ".ogg", ".m4a", ".wav", ".wma"} {
		path := filepath.Join(dir, "x"+ext)
		mustWrite(t, path, []byte("raw audio"))
		if err := WriteTags(path, &Metadata{Title: "t"}); err != nil {
			t.Errorf("%s: %v", ext, err)
		}
		if got, _ := os.ReadFile(path); !bytes.Equal(got, []byte("raw audio")) {
			t.Errorf("%s was rewritten", ext)
		}
	}
}

func TestWriteTagsNilIsANoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.mp3")
	mustWrite(t, path, []byte("raw audio"))
	if err := WriteTags(path, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, []byte("raw audio")) {
		t.Error("WriteTags(nil) rewrote the file")
	}
}

// flacBlockFor builds one file block: header byte plus size, then the payload.
func flacBlockFor(typ int, payload []byte, last bool) []byte {
	h := byte(typ)
	if last {
		h |= 0x80
	}
	return append([]byte{h, byte(len(payload) >> 16), byte(len(payload) >> 8), byte(len(payload))}, payload...)
}

// flacRead parses a flac byte buffer into its blocks and the audio that follows.
func flacRead(t *testing.T, b []byte) ([][]byte, []byte) {
	t.Helper()
	var blocks [][]byte
	off := 4
	for off+4 <= len(b) {
		flag := b[off]
		size := int(b[off+1])<<16 | int(b[off+2])<<8 | int(b[off+3])
		if off+4+size > len(b) {
			t.Fatalf("block %d overruns the file", len(blocks))
		}
		blocks = append(blocks, b[off:off+4+size])
		off += 4 + size
		if flag&0x80 != 0 {
			break
		}
	}
	return blocks, b[off:]
}

// TestFlacVorbis pins the block layout: little-endian counts and lengths, then
// the KEY=value pairs. Vorbis is big-endian elsewhere, so the endianness is the
// thing worth pinning.
func TestFlacVorbis(t *testing.T) {
	b := flacVorbis(&Metadata{
		Title: "晴天", Artist: "周杰伦", Album: "叶惠美", Track: "3/12", Year: "2003", Genre: "流行",
	})
	n := binary.LittleEndian.Uint32(b)
	if n != 6 {
		t.Fatalf("comment count = %d, want 6", n)
	}
	got := map[string]string{}
	off := 4
	for i := uint32(0); i < n; i++ {
		l := int(binary.LittleEndian.Uint32(b[off:]))
		off += 4
		k, v, ok := strings.Cut(string(b[off:off+l]), "=")
		off += l
		if !ok {
			t.Fatalf("comment %d is not KEY=value: %q", i, b[off-l:off])
		}
		got[k] = v
	}
	if off != len(b) {
		t.Errorf("the block is %d bytes but the comments occupy %d", len(b), off)
	}
	for k, want := range map[string]string{
		"TITLE": "晴天", "ARTIST": "周杰伦", "ALBUM": "叶惠美",
		"TRACKNUMBER": "3/12", "DATE": "2003", "GENRE": "流行",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, want %q", k, got[k], want)
		}
	}
	if got := flacVorbis(&Metadata{}); got != nil {
		t.Error("an empty Metadata produced a vorbis comment")
	}
	// A lyric lands as its own comment.
	b = flacVorbis(&Metadata{Lyric: "[00:01.000]晴 天"})
	if n := binary.LittleEndian.Uint32(b); n != 1 {
		t.Fatalf("a lyric produced %d comments, want 1", n)
	}
	if !strings.HasSuffix(string(b), "[00:01.000]晴 天") {
		t.Errorf("the lyric text is not in the block: %q", b)
	}
}

// TestFlacPictureBlock pins the PICTURE fields: big-endian, type 3, and the
// dimensions parsed out of the JPEG.
func TestFlacPictureBlock(t *testing.T) {
	pic := jpegHeader(320, 240)
	b := flacPicture(&Metadata{Picture: pic})
	if len(b) < 42+len(pic) {
		t.Fatalf("block is too short: %d bytes", len(b))
	}
	if got := binary.BigEndian.Uint32(b[0:]); got != 3 {
		t.Errorf("picture type = %d, want 3 (front cover)", got)
	}
	if got := binary.BigEndian.Uint32(b[4:]); got != 10 {
		t.Errorf("mime length = %d, want 10", got)
	}
	if got := string(b[8:18]); got != "image/jpeg" {
		t.Errorf("mime = %q", got)
	}
	if got := binary.BigEndian.Uint32(b[18:]); got != 0 {
		t.Errorf("description length = %d, want 0", got)
	}
	if w := binary.BigEndian.Uint32(b[22:]); w != 320 {
		t.Errorf("width = %d, want 320", w)
	}
	if h := binary.BigEndian.Uint32(b[26:]); h != 240 {
		t.Errorf("height = %d, want 240", h)
	}
	if got := binary.BigEndian.Uint32(b[30:]); got != 24 {
		t.Errorf("depth = %d, want 24", got)
	}
	if got := binary.BigEndian.Uint32(b[34:]); got != 0 {
		t.Errorf("colors = %d, want 0", got)
	}
	if got := binary.BigEndian.Uint32(b[38:]); got != uint32(len(pic)) {
		t.Errorf("data length = %d, want %d", got, len(pic))
	}
	if !bytes.Equal(b[42:], pic) {
		t.Error("the picture bytes were not preserved")
	}
	if got := flacPicture(&Metadata{}); got != nil {
		t.Error("no picture produced a picture block")
	}
	// An image whose size cannot be read still writes a valid block.
	b = flacPicture(&Metadata{Picture: []byte("not a jpeg")})
	if b == nil {
		t.Fatal("no picture block for an unrecognised image")
	}
	if got := binary.BigEndian.Uint32(b[22:]); got != 0 {
		t.Errorf("an unrecognised image wrote width %d, want 0", got)
	}
}

// TestWriteFlacTagsRoundTrip writes a minimal flac by hand, tags it, and reads
// the blocks back. STREAMINFO must stay first and untouched, and the last-block
// bit must move from it to the block that now ends the table.
func TestWriteFlacTagsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "晴天.flac")
	stream := append([]byte{0x66, 0x6c, 0x61, 0x63}, 0x28, 0x00, 0x28, 0x00)
	stream = append(stream, 0x00, 0x01, 0x86, 0x90)
	stream = append(stream, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10)
	stream = append(stream, 0x00, 0x44, 0xac, 0x00, 0x00, 0x00, 0x20, 0x00)
	stream = append(stream, bytes.Repeat([]byte{0x00}, 16)...)
	if len(stream) != 42 {
		t.Fatalf("a streaminfo block is %d bytes, want 42", len(stream))
	}
	audio := []byte{0x01, 0xff, 0xff, 0xff}
	body := append(flacBlockFor(flacStreamInfo, stream, true), audio...)
	mustWrite(t, path, append([]byte{'f', 'L', 'a', 'C'}, body...))

	if err := WriteTags(path, &Metadata{
		Title: "晴天", Artist: "周杰伦", Album: "叶惠美",
		Picture: []byte{0xff, 0xd8, 0xff, 0xe0, 1, 2, 3, 4},
	}); err != nil {
		t.Fatalf("WriteTags: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("fLaC")) {
		t.Fatal("the fLaC magic was lost")
	}
	blocks, rest := flacRead(t, got)
	if len(blocks) != 3 {
		t.Fatalf("blocks = %d, want 3", len(blocks))
	}
	types := []int{int(blocks[0][0]) & 0x7f, int(blocks[1][0]) & 0x7f, int(blocks[2][0]) & 0x7f}
	if types[0] != flacStreamInfo || types[1] != flacVorbisComment || types[2] != flacPictureBlock {
		t.Fatalf("block types = %v", types)
	}
	if !bytes.Equal(blocks[0][4:], stream) {
		t.Error("the streaminfo block was modified")
	}
	if blocks[0][0]&0x80 != 0 {
		t.Error("streaminfo still claims to be the last block")
	}
	if blocks[1][0]&0x80 != 0 {
		t.Error("the comment claims to be the last block")
	}
	if blocks[2][0]&0x80 == 0 {
		t.Error("the picture does not carry the last flag")
	}
	if !bytes.Equal(rest, audio) {
		t.Error("the audio frames were not preserved")
	}

	// A second write replaces the comment instead of adding a third block.
	if err := WriteTags(path, &Metadata{Title: "新标题"}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(path)
	blocks, rest = flacRead(t, got)
	if len(blocks) != 2 {
		t.Fatalf("re-tagged blocks = %d, want 2", len(blocks))
	}
	if blocks[0][0]&0x7f != flacStreamInfo || blocks[1][0]&0x7f != flacVorbisComment {
		t.Errorf("re-tagged block types = %d, %d", blocks[0][0]&0x7f, blocks[1][0]&0x7f)
	}
	if !bytes.Equal(blocks[0][4:], stream) {
		t.Error("the re-tag modified streaminfo")
	}
	if !bytes.HasSuffix(rest, audio) {
		t.Error("the re-tag lost the audio")
	}
}

// TestWriteFlacTagsKeepsForeignPins the blocks a real file already has: padding,
// seektable, application data. Only the comment and the picture come out.
func TestWriteFlacTagsKeepsForeign(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.flac")
	stream := bytes.Repeat([]byte{0xaa}, 12)
	buf := []byte{'f', 'L', 'a', 'C'}
	for _, bl := range [][]byte{
		flacBlockFor(flacStreamInfo, stream, false),
		flacBlockFor(flacVorbisComment, bytes.Repeat([]byte{0xbb}, 20), false),
		flacBlockFor(flacPadding, bytes.Repeat([]byte{0x00}, 8), false),
		flacBlockFor(flacMetadataBlock, bytes.Repeat([]byte{0xcc}, 4), false),
		flacBlockFor(flacApplication, bytes.Repeat([]byte{0xdd}, 6), false),
		flacBlockFor(flacVorbisComment, bytes.Repeat([]byte{0xbb}, 20), false),
		flacBlockFor(flacPictureBlock, bytes.Repeat([]byte{0xee}, 10), true),
	} {
		buf = append(buf, bl...)
	}
	buf = append(buf, 0x01, 0xff)
	mustWrite(t, path, buf)

	if err := WriteTags(path, &Metadata{Title: "t", Artist: "a"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	blocks, rest := flacRead(t, got)
	count := map[int]int{}
	for _, b := range blocks {
		count[int(b[0])&0x7f]++
	}
	if count[flacVorbisComment] != 1 {
		t.Errorf("comment blocks = %d, want 1 (the old ones were dropped)", count[flacVorbisComment])
	}
	if count[flacPictureBlock] != 0 {
		t.Errorf("picture blocks = %d, want 0", count[flacPictureBlock])
	}
	for _, typ := range []int{flacStreamInfo, flacPadding, flacMetadataBlock, flacApplication} {
		if count[typ] != 1 {
			t.Errorf("block %d: %d copies, want 1", typ, count[typ])
		}
	}
	if !bytes.HasSuffix(rest, []byte{0x01, 0xff}) {
		t.Error("the audio frames were not preserved")
	}
}

func TestWriteFlacTagsRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string][]byte{
		"not flac":   []byte("not flac at all"),
		"magic only": []byte("fLaC"),
		"bad magic":  {0x00, 'f', 'L', 'a', 'C'},
		"short block": {
			'f', 'L', 'a', 'C', 0x80, 0x00, 0x00, 0x0a, 0x00, 0x00,
		},
	} {
		path := filepath.Join(dir, "bad.flac")
		mustWrite(t, path, body)
		if err := WriteTags(path, &Metadata{Title: "t"}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// TestFlacBlockLength pins the 24-bit length field: a payload too large for it is
// dropped rather than truncated, which would corrupt the whole block table.
func TestFlacBlockLength(t *testing.T) {
	if got := flacBlock(flacVorbisComment, nil); got != nil {
		t.Error("an empty payload produced a block")
	}
	if got := flacBlock(flacVorbisComment, bytes.Repeat([]byte{0}, 0xffffff+1)); got != nil {
		t.Errorf("a %d byte payload fit in a 24-bit length field", 0xffffff+1)
	}
	b := flacBlock(flacVorbisComment, []byte{1, 2, 3, 4, 5})
	if b[0] != flacVorbisComment || b[1] != 0 || b[2] != 0 || b[3] != 5 {
		t.Errorf("block header = % x", b[:4])
	}
	if !bytes.Equal(b[4:], []byte{1, 2, 3, 4, 5}) {
		t.Error("the payload was not preserved")
	}
}

func TestDetectImageMime(t *testing.T) {
	for _, tc := range []struct {
		body []byte
		want string
	}{
		{[]byte{0xff, 0xd8, 0xff, 0xe0, 1, 2}, "image/jpeg"},
		{[]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0}, "image/png"},
		{[]byte("RIFF\x04\x00\x00\x00WEBP"), "image/webp"},
		{[]byte("GIF89aabc"), "image/gif"},
		{[]byte("GIF87aabc"), "image/gif"},
		{[]byte("BMabc"), "image/bmp"},
		{[]byte("not an image"), "image/jpeg"},
		{[]byte{}, "image/jpeg"},
		{[]byte{0xff, 0xd8}, "image/jpeg"},                 // too short to be real, still jpeg-shaped
		{[]byte("RIFF\x04\x00\x00\x00AVI "), "image/jpeg"}, // a RIFF that is not a webp
	} {
		if got := DetectImageMime(tc.body); got != tc.want {
			t.Errorf("DetectImageMime(% x) = %q, want %q", tc.body[:minInt(len(tc.body), 8)], got, tc.want)
		}
	}
}

func TestJpegSize(t *testing.T) {
	for _, tc := range []struct {
		body []byte
		w, h int
	}{
		{jpegHeader(640, 480), 640, 480},
		{jpegHeader(1920, 1080), 1920, 1080},
		{jpegHeader(1, 1), 1, 1},
		{jpegHeader(0xffff, 0xffff), 65535, 65535},
		{[]byte{0xff, 0xd8, 0xff, 0xe0, 0, 0}, 0, 0},
		{[]byte{0xff, 0xd9}, 0, 0},
		{[]byte{0x50, 0x4e, 0x47}, 0, 0},
		{[]byte{0xff, 0xd8, 0xff, 0xc0}, 0, 0}, // a SOF with no payload
		{[]byte{0xff, 0xd8, 0xff, 0xda, 0x00}, 0, 0},
		{append(jpegHeader(100, 50), jpegHeader(200, 60)...), 100, 50}, // the first SOF wins
	} {
		w, h := jpegSize(tc.body)
		if w != tc.w || h != tc.h {
			t.Errorf("jpegSize = %dx%d, want %dx%d", w, h, tc.w, tc.h)
		}
	}
}

// jpegHeader builds a minimal baseline JPEG with an SOF0 at the given size.
func jpegHeader(w, h int) []byte {
	b := []byte{0xff, 0xd8}               // SOI
	b = append(b, 0xff, 0xe0, 0x00, 0x10) // APP0, 16 bytes of payload
	b = append(b, 'J', 'F', 'I', 'F', 0, 0, 0x01, 0x01, 0, 0, 0, 0, 0, 0, 0, 0)
	b = append(b, 0xff, 0xdb, 0x00, 0x43) // DQT
	b = append(b, bytes.Repeat([]byte{0x10}, 65)...)
	b = append(b, 0xff, 0xc0, 0x00, 0x11, 0x08, byte(h>>8), byte(h), byte(w>>8), byte(w))
	b = append(b, 0x03, 0x01, 0x22, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01)
	return b
}
