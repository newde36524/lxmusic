// tags.go writes title/artist/album/cover/lyric into a downloaded audio file.
//
// The desktop app does this with node-id3 (mp3) and FlacProcessor (flac); both
// are npm packages, and pulling node in would end the CGO-free single binary.
// These are the frames and blocks those two write for the fields this CLI has,
// implemented from the format specs instead. ID3v2.4 is written — the version
// node-id3 emits by default, and the one that made ID3v2.3's synchsafe frame
// sizes obsolete — with UTF-16 text. FLAC gets a VORBIS_COMMENT and a PICTURE
// block, with STREAMINFO and every other existing block preserved in place.
package lx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// Metadata is the tag payload.
type Metadata struct {
	Title         string
	Artist        string
	Album         string
	Track         string // "3" or "3/12"
	Year          string
	Genre         string
	Lyric         string // LRC text
	Picture       []byte
	PictureMime   string // sniffed from Picture when empty
	PictureWidth  int    // 0 = as large as the frame is
	PictureHeight int
}

// WriteTags embeds m into the file at path, keyed on its extension. Only mp3 and
// flac are written — the desktop app is equally picky, and a silent no-op is the
// right answer for a format this file does not know.
func WriteTags(path string, m *Metadata) error {
	if m == nil {
		return nil
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return writeMP3Tags(path, m)
	case ".flac":
		return writeFlacTags(path, m)
	default:
		return nil
	}
}

// DetectImageMime sniffs the bytes instead of trusting the URL: the extension a
// cover is served under is often wrong, and a player that takes the tag's mime
// at face value will refuse to draw it.
func DetectImageMime(b []byte) string {
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "image/jpeg"
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return "image/png"
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "image/webp"
	case len(b) >= 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))):
		return "image/gif"
	case len(b) >= 2 && bytes.Equal(b[:2], []byte("BM")):
		return "image/bmp"
	}
	return "image/jpeg"
}

// jpegSize reads a baseline JPEG's dimensions from its SOF marker. Only that
// one family is parsed, since it is the cover this CLI actually meets; anything
// else leaves both dimensions at 0, and a 0/0 PICTURE block is valid.
func jpegSize(b []byte) (w, h int) {
	if len(b) < 4 || !(b[0] == 0xff && b[1] == 0xd8) {
		return 0, 0
	}
	i := 2
	for i+4 <= len(b) {
		if b[i] != 0xff {
			i++
			continue
		}
		marker := b[i+1]
		if marker == 0xd9 || marker == 0xda { // end of image, start of scan
			break
		}
		if (marker >= 0xd0 && marker <= 0xd7) || marker == 0x01 || marker == 0xd8 {
			i += 2
			continue // markers that carry no length
		}
		if i+4 > len(b) {
			break
		}
		n := int(b[i+2])<<8 | int(b[i+3])
		if n < 2 || i+2+n > len(b) {
			break
		}
		if marker >= 0xc0 && marker <= 0xcf && marker != 0xc4 && marker != 0xc8 && marker != 0xcc {
			if i+2+n >= i+9 {
				return int(b[i+7])<<8 | int(b[i+8]), int(b[i+5])<<8 | int(b[i+6])
			}
		}
		i += 2 + n
	}
	return 0, 0
}

// -------------------- ID3v2.4 --------------------

func utf16LE(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u))
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return b
}

func id3Text(s string) []byte {
	b := []byte{0x01}         // UTF-16
	b = append(b, 0xff, 0xfe) // BOM
	b = append(b, utf16LE(s)...)
	return append(b, 0x00, 0x00) // terminator, in the encoding's own width
}

func id3Frame(id string, data []byte) []byte {
	b := []byte(id)
	b = append(b, byte(len(data)>>24), byte(len(data)>>16), byte(len(data)>>8), byte(len(data)))
	b = append(b, 0x00, 0x00) // flags
	return append(b, data...)
}

// id3Tag builds the whole tag, or nil when there is nothing to write.
func id3Tag(m *Metadata) []byte {
	if m == nil {
		return nil
	}
	var frames bytes.Buffer
	for _, f := range [][2]string{
		{"TIT2", m.Title}, {"TPE1", m.Artist}, {"TALB", m.Album},
		{"TRCK", m.Track}, {"TDRC", m.Year}, {"TCON", m.Genre},
	} {
		if f[1] == "" {
			continue
		}
		frames.Write(id3Frame(f[0], id3Text(f[1])))
	}
	if m.Lyric != "" {
		b := []byte{0x01, 0xff, 0xfe}
		b = append(b, []byte("zho")...) // ISO 639-2 language: Chinese
		b = append(b, 0x00, 0x00)       // empty content description
		b = append(b, utf16LE(m.Lyric)...)
		frames.Write(id3Frame("USLT", b))
	}
	if len(m.Picture) > 0 {
		mime := m.PictureMime
		if mime == "" {
			mime = DetectImageMime(m.Picture)
		}
		b := []byte{0x00} // ISO-8859-1 for the mime and the description
		b = append(b, mime...)
		b = append(b, 0x00) // mime terminator
		b = append(b, 0x03) // picture type 3: cover, front
		b = append(b, 0x00) // empty description
		b = append(b, m.Picture...)
		frames.Write(id3Frame("APIC", b))
	}
	if frames.Len() == 0 {
		return nil
	}
	out := []byte{'I', 'D', '3', 0x04, 0x00, 0x00} // major 2.4, revision 0, flags 0
	out = append(out, synchsafe(frames.Len())...)
	return append(out, frames.Bytes()...)
}

// synchsafe encodes 28 bits in four bytes. Only the tag header uses it; v2.4
// frame sizes are ordinary big-endian integers.
func synchsafe(n int) []byte {
	return []byte{byte(n >> 21 & 0x7f), byte(n >> 14 & 0x7f), byte(n >> 7 & 0x7f), byte(n & 0x7f)}
}

// stripID3v2 drops an ID3v2 header so a re-tag does not leave two tags in one
// file. Only v2.2 through v2.4 are recognised; a file that opens with "ID3" and
// is not one of those is left untouched rather than mangled.
func stripID3v2(data []byte) []byte {
	if len(data) < 9 || !bytes.Equal(data[:3], []byte("ID3")) {
		return data
	}
	var head, size int
	switch data[3] {
	case 4, 3:
		if len(data) < 10 {
			return data
		}
		head = 10
		size = int(data[6])<<21 | int(data[7])<<14 | int(data[8])<<7 | int(data[9])
		if data[5]&0x10 != 0 {
			head += 10 // footer, flagged in the header's flags byte
		}
	case 2:
		// v2.2's header is one byte shorter and its size is plain, not synchsafe
		head = 9
		size = int(data[6])<<16 | int(data[7])<<8 | int(data[8])
	default:
		return data
	}
	if head+size <= len(data) {
		return data[head+size:]
	}
	return data
}

// -------------------- FLAC --------------------

const (
	flacStreamInfo    = 0
	flacPadding       = 1
	flacMetadataBlock = 2
	flacApplication   = 3
	flacVorbisComment = 4
	flacPictureBlock  = 6
)

func le32(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

// flacBlock builds one block header plus payload. A 24-bit length field cannot
// name anything over 16 MiB, so an oversized payload is dropped rather than
// truncated in flight.
func flacBlock(t int, payload []byte) []byte {
	if len(payload) == 0 || len(payload) > 0xffffff {
		return nil
	}
	b := []byte{byte(t), byte(len(payload) >> 16), byte(len(payload) >> 8), byte(len(payload))}
	return append(b, payload...)
}

func flacVorbis(m *Metadata) []byte {
	var cs []string
	put := func(k, v string) {
		if v != "" {
			cs = append(cs, k+"="+v)
		}
	}
	put("TITLE", m.Title)
	put("ARTIST", m.Artist)
	put("ALBUM", m.Album)
	put("TRACKNUMBER", m.Track)
	put("DATE", m.Year)
	put("GENRE", m.Genre)
	put("LYRICS", m.Lyric)
	if len(cs) == 0 {
		return nil
	}
	var b []byte
	b = append(b, le32(uint32(len(cs)))...)
	for _, c := range cs {
		b = append(b, le32(uint32(len(c)))...)
		b = append(b, c...)
	}
	return b
}

func flacPicture(m *Metadata) []byte {
	if len(m.Picture) == 0 {
		return nil
	}
	mime := m.PictureMime
	if mime == "" {
		mime = DetectImageMime(m.Picture)
	}
	w, h := m.PictureWidth, m.PictureHeight
	if w == 0 && h == 0 {
		w, h = jpegSize(m.Picture)
	}
	be := func(n uint32) []byte {
		return []byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
	}
	ws := func(s string) []byte { return append(be(uint32(len(s))), s...) }
	b := []byte{}
	b = append(b, be(3)...) // picture type 3: cover, front
	b = append(b, ws(mime)...)
	b = append(b, ws("")...)                                  // description
	for _, n := range []uint32{uint32(w), uint32(h), 24, 0} { // width, height, depth, colors
		b = append(b, be(n)...)
	}
	b = append(b, be(uint32(len(m.Picture)))...)
	b = append(b, m.Picture...)
	return b
}

// -------------------- writers --------------------

func writeMP3Tags(path string, m *Metadata) error {
	tag := id3Tag(m)
	if tag == nil {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	body = stripID3v2(body)
	if len(body) == 0 {
		return fmt.Errorf("no audio data")
	}
	return os.WriteFile(path, append(tag, body...), 0o644)
}

func writeFlacTags(path string, m *Metadata) error {
	var blocks [][]byte
	if b := flacBlock(flacVorbisComment, flacVorbis(m)); b != nil {
		blocks = append(blocks, b)
	}
	if b := flacBlock(flacPictureBlock, flacPicture(m)); b != nil {
		blocks = append(blocks, b)
	}
	if len(blocks) == 0 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 4 || !bytes.Equal(data[:4], []byte("fLaC")) {
		return fmt.Errorf("no fLaC header")
	}
	// Keep every block except the comment and picture, which are replaced.
	off := 4
	keep := [][]byte(nil)
	for {
		if off+4 > len(data) {
			return fmt.Errorf("truncated flac metadata")
		}
		flag := data[off]
		size := int(data[off+1])<<16 | int(data[off+2])<<8 | int(data[off+3])
		if off+4+size > len(data) {
			return fmt.Errorf("truncated flac metadata")
		}
		if data[off]&0x7f != flacVorbisComment && data[off]&0x7f != flacPictureBlock {
			keep = append(keep, append([]byte(nil), data[off:off+4+size]...))
		}
		off += 4 + size
		if flag&0x80 != 0 {
			break
		}
	}
	if len(keep) == 0 {
		return fmt.Errorf("no flac streaminfo")
	}
	all := append(keep, blocks...)
	out := []byte{'f', 'L', 'a', 'C'}
	for i, b := range all {
		b[0] &= 0x7f // re-mark the last block; only one may carry the bit
		if i == len(all)-1 {
			b[0] |= 0x80
		}
		out = append(out, b...)
	}
	out = append(out, data[off:]...)
	return os.WriteFile(path, out, 0o644)
}
