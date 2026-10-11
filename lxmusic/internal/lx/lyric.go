package lx

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
)

/*
KRC is kugou's synced lyric format. Renderer module 92617 shows the transport is
base64 of a 4-byte "krc1" magic followed by an XOR-encrypted zlib stream, and the
decrypted text is *not* an LRC: it carries "[start,dur]" line timestamps in
milliseconds, per-word opening "<rel,dur,0>" markers with no closing tag, and an
optional "[language:base64-json]" line holding the romanized and the translated
lyric as arrays indexed by lyric line. The decoder below reproduces that pipeline.

Kuwo's mlyric.kuwo.cn answer is two header lines followed by a raw zlib stream,
which is what the app inflates before parsing.
*/

const krcMagic = "krc1"

// krcKey is the 16-byte XOR key, verbatim from renderer.js module 92617.
var krcKey = []byte{64, 71, 97, 119, 94, 50, 116, 71, 81, 54, 49, 45, 206, 210, 110, 105}

var (
	// krcIDHead strips the "[id:$...]" head. Kugou's ids are colon-delimited
	// ("[id:$kugou:lyric:hash:abc]"), which the bundle's "\w+" form cannot
	// reach, so the id is matched as "anything but ]" instead.
	krcIDHead = regexp.MustCompile(`^.*\[id:\$[^\]]+\]\n`)
	// krcLanguage is the base64 JSON holding the romanized and translated lyric.
	krcLanguage = regexp.MustCompile(`\[language:([\w=\\/+]+)\]`)
	// krcTimestamp matches "[start,dur]": the line's absolute start and its
	// duration, both in milliseconds.
	krcTimestamp = regexp.MustCompile(`\[(\d+),\d+\]`)
	// krcWordSpan matches a per-word marker. Words carry an opening marker only
	// - "<start,dur,0>" or "<start,dur>" - and never a closing one.
	krcWordSpan = regexp.MustCompile(`<\d+,\d+(?:,\d+)?` + `>`)
	krcFirstTag = regexp.MustCompile(`\[[\d.:]+\]`)
)

// krcLangEntry is one element of the base64 JSON inside "[language:...]".
// lyricContent is an array indexed by lyric line; the last entry of each type
// wins.
type krcLangEntry struct {
	Type         int `json:"type"`
	LyricContent any `json:"lyricContent"`
}

// DecodeKugouLyric turns the `content` field of lyrics.kugou.com/download into a
// Lyric. LRC payloads are base64 text; KRC payloads are base64 "krc1" + XOR +
// zlib, and the decrypted text is parsed into structured fields.
func DecodeKugouLyric(content string, isKRC bool) *Lyric {
	plain, isKRCFile := decodeKugouPayload(content)
	l := &Lyric{Plain: strings.TrimSpace(plain), Raw: content}
	if isKRCFile {
		l.Format = "krc"
		if d := parseKRC(plain); d != nil {
			d.Raw = content
			return d
		}
		return l
	}
	l.Format = "lrc"
	return l
}

// decodeKugouPayload base64-decodes the payload and, when it is a KRC file,
// strips the magic, XORs the ciphertext and inflates the zlib stream.
func decodeKugouPayload(content string) (string, bool) {
	s := strings.TrimSpace(content)
	if s == "" {
		return "", false
	}
	if strings.HasPrefix(s, krcMagic) { // already decrypted
		return s, true
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return strings.TrimSpace(s), false
	}
	if len(raw) < 4 || string(raw[:4]) != krcMagic {
		return strings.TrimSpace(string(raw)), false
	}
	ct := append([]byte(nil), raw[4:]...)
	for i := range ct {
		ct[i] ^= krcKey[i%len(krcKey)]
	}
	z, err := zlib.NewReader(bytes.NewReader(ct))
	if err != nil {
		return strings.TrimSpace(string(raw)), false
	}
	defer z.Close()
	out, err := io.ReadAll(z)
	if err != nil {
		return strings.TrimSpace(string(raw)), false
	}
	return string(out), true
}

// parseKRC turns decrypted KRC text into a Lyric.
//
// KRC is not an LRC. Each lyric line is "[start,dur]<rel,dur,0>字<rel,dur,0>字":
// start and dur are the line's absolute and relative positions in milliseconds
// and every word carries an opening marker only - there is no closing tag.
// Metadata lines ([ti:], [ar:], [al:], [offset:] ...) carry no timestamp.
func parseKRC(src string) *Lyric {
	e := strings.ReplaceAll(src, "\r", "")
	if m := krcIDHead.FindStringIndex(e); m != nil {
		e = e[m[1]:]
	}

	// The language block holds the romanized (type 0) and translated (type 1)
	// lyric. Drop the whole line including its newline, as the bundle does.
	var roman, trans []string
	if m := krcLanguage.FindStringSubmatchIndex(e); m != nil {
		lang := e[m[2]:m[3]]
		end := m[1]
		if end < len(e) && e[end] == '\n' {
			end++
		}
		e = e[:m[0]] + e[end:]
		if b, err := base64.StdEncoding.DecodeString(lang); err == nil {
			var doc struct {
				Content []krcLangEntry `json:"content"`
			}
			if err := json.Unmarshal(b, &doc); err == nil {
				for _, c := range doc.Content {
					switch c.Type {
					case 0:
						roman = krcLangLines(c.LyricContent)
					case 1:
						trans = krcLangLines(c.LyricContent)
					}
				}
			}
		}
	}

	// Rewrite "[start,dur]" into an LRC tag. Remember the tags, one per lyric
	// line, for the romanized and translated variants.
	var b strings.Builder
	var tags []string
	for _, line := range strings.Split(e, "\n") {
		if m := krcTimestamp.FindStringSubmatchIndex(line); m != nil {
			tag := krcTag(krcInt(line[m[2]:m[3]]))
			tags = append(tags, tag)
			line = line[:m[0]] + tag + line[m[1]:]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	plain := strings.TrimRight(b.String(), "\n")

	for i, tag := range tags {
		if i < len(roman) {
			roman[i] = tag + roman[i]
		}
		if i < len(trans) {
			trans[i] = tag + trans[i]
		}
	}

	return &Lyric{
		Plain:  strings.TrimSpace(stripKRCSpans(html.UnescapeString(plain))),
		Roman:  strings.TrimSpace(html.UnescapeString(strings.Join(roman, "\n"))),
		Trans:  strings.TrimSpace(html.UnescapeString(strings.Join(trans, "\n"))),
		Format: "krc",
	}
}

// stripKRCSpans removes the per-word "<rel,dur,0>" markers.
func stripKRCSpans(s string) string {
	return krcWordSpan.ReplaceAllString(s, "")
}

// krcLangLines flattens one lyricContent entry to a per-line slice.
func krcLangLines(v any) []string {
	switch x := v.(type) {
	case []any:
		out := make([]string, len(x))
		for i, e := range x {
			if s, ok := e.(string); ok {
				out[i] = s
			}
		}
		return out
	case string:
		return []string{x}
	}
	return nil
}

// krcInt parses a millisecond field, tolerating the parseInt-style truncation
// the bundle relies on.
func krcInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// krcTag renders milliseconds as a bracketed LRC time tag, "[mm:ss.xx]".
func krcTag(ms int) string {
	if ms < 0 {
		ms = 0
	}
	sec := ms / 1000
	return fmt.Sprintf("[%02d:%02d.%02d]", sec/60, sec%60, (ms%1000)/10)
}

// FormatLRCTime renders milliseconds as mm:ss.xx.
func FormatLRCTime(ms int64) string {
	sec := ms / 1000
	return fmt.Sprintf("%02d:%02d.%02d", sec/60, sec%60, (ms%1000)/10)
}

// ParseLRCLine pulls the timestamp out of a "[mm:ss.xx] text" line.
func ParseLRCLine(line string) (ms int64, text string) {
	line = strings.TrimSpace(line)
	tag := krcFirstTag.FindString(line)
	if tag == "" {
		return -1, line
	}
	tok := tag[1 : len(tag)-1]
	parts := strings.Split(tok, ":")
	if len(parts) != 2 {
		return -1, line
	}
	min, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1, line
	}
	// Split the seconds and the fraction apart, then normalize the fraction to
	// milliseconds: LRC writes two digits (10ms units) but some encoders emit
	// three.
	secStr, frac := parts[1], ""
	if i := strings.IndexByte(parts[1], '.'); i >= 0 {
		secStr, frac = parts[1][:i], parts[1][i+1:]
	}
	for len(frac) < 3 {
		frac += "0"
	}
	sec, err := strconv.Atoi(secStr)
	if err != nil {
		return -1, line
	}
	fracMS, err := strconv.Atoi(frac[:3])
	if err != nil {
		return -1, line
	}
	return int64(min)*60000 + int64(sec)*1000 + int64(fracMS), strings.TrimSpace(line[len(tag):])
}
