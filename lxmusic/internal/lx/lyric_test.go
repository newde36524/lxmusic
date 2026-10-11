package lx

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"strings"
	"testing"
)

// krcEncode builds the wire form of a KRC document the way kugou sends it:
// zlib the text, XOR it with the fixed 16-byte key, prefix the krc1 magic and
// base64 the whole thing.
func krcEncode(t *testing.T, text string) string {
	t.Helper()
	var buf bytes.Buffer
	z := zlib.NewWriter(&buf)
	if _, err := z.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	ct := buf.Bytes()
	for i := range ct {
		ct[i] ^= krcKey[i%len(krcKey)]
	}
	ct = append([]byte(krcMagic), ct...)
	return base64.StdEncoding.EncodeToString(ct)
}

// krcFixture mirrors a real kugou document: metadata tags, a base64 language
// block, then lyric lines of "[start,dur]" plus per-word opening markers.
// 晴天 at 0ms and at 29264ms.
const krcFixture = `[ti:晴天]
[ar:周杰伦]
[al:叶惠美]
[by:]
[offset:0]
[language:PLACEHOLDER]
[0,2250]<0,160,0>晴<160,160,0>天
[29264,3446]<0,390,0>故<390,392,0>事<782,448,0>的小<2152,374,0>黄<2526,504,0>花`

const krcLangJSON = `{"content":[` +
	`{"type":0,"lyricContent":["hello","world"]},` +
	`{"type":1,"lyricContent":["first line","second line"]}]}`

// TestDecodeKugouLyricKRC round-trips the decoder through the real transport.
// It guards three things: the timestamp is the line's absolute start in
// milliseconds, the brackets are written exactly once, and the per-word markers
// are stripped.
func TestDecodeKugouLyricKRC(t *testing.T) {
	doc := strings.Replace(krcFixture, "PLACEHOLDER", base64.StdEncoding.EncodeToString([]byte(krcLangJSON)), 1)
	wire := krcEncode(t, doc)

	ly := DecodeKugouLyric(wire, true)
	if ly == nil {
		t.Fatal("nil lyric")
	}
	if ly.Format != "krc" {
		t.Errorf("format = %q, want krc", ly.Format)
	}
	if ly.Raw != wire {
		t.Errorf("raw was dropped on the krc path")
	}
	wantPlain := "[ti:晴天]\n[ar:周杰伦]\n[al:叶惠美]\n[by:]\n[offset:0]\n" +
		"[00:00.00]晴天\n[00:29.26]故事的小黄花"
	if ly.Plain != wantPlain {
		t.Errorf("plain:\n got %q\nwant %q", ly.Plain, wantPlain)
	}
	if strings.Contains(ly.Plain, "[[") {
		t.Errorf("double-bracketed timestamp leaked into plain: %q", ly.Plain)
	}
	if strings.Contains(ly.Plain, "<") {
		t.Errorf("KRC word markers leaked into plain: %q", ly.Plain)
	}
	if got := ly.Plain; strings.Contains(got, "PLACEHOLDER") {
		t.Errorf("language block leaked into plain: %q", got)
	}
	wantRoman := "[00:00.00]hello\n[00:29.26]world"
	if ly.Roman != wantRoman {
		t.Errorf("roman:\n got %q\nwant %q", ly.Roman, wantRoman)
	}
	wantTrans := "[00:00.00]first line\n[00:29.26]second line"
	if ly.Trans != wantTrans {
		t.Errorf("trans:\n got %q\nwant %q", ly.Trans, wantTrans)
	}
}

// TestParseKRCStripsIDHead pins the "[id:$...]" head strip, which only fires
// when that tag is on the first line.
func TestParseKRCStripsIDHead(t *testing.T) {
	src := "[id:$kugou:lyric:1:abc123]\n[ti:晴天]\n[0,2250]<0,160,0>晴<160,160,0>天"
	ly := parseKRC(src)
	if ly == nil {
		t.Fatal("nil lyric")
	}
	if strings.Contains(ly.Plain, "id:$") {
		t.Errorf("id head not stripped: %q", ly.Plain)
	}
	if strings.Contains(ly.Plain, "[id:") {
		t.Errorf("id head leaked: %q", ly.Plain)
	}
	want := "[ti:晴天]\n[00:00.00]晴天"
	if ly.Plain != want {
		t.Errorf("plain:\n got %q\nwant %q", ly.Plain, want)
	}
}

// TestParseKRCLastEntryWins pins that the last language entry of each type wins.
func TestParseKRCLastEntryWins(t *testing.T) {
	lang := `{"content":[` +
		`{"type":0,"lyricContent":["stale"]},` +
		`{"type":0,"lyricContent":["fresh"]},` +
		`{"type":1,"lyricContent":["only one translation"]}]}`
	src := "[ti:测试]\n[language:" + base64.StdEncoding.EncodeToString([]byte(lang)) + `]
[0,1000]<0,500,0>a<500,500,0>b`
	ly := parseKRC(src)
	if ly == nil {
		t.Fatal("nil lyric")
	}
	if ly.Roman != "[00:00.00]fresh" {
		t.Errorf("roman = %q", ly.Roman)
	}
	if ly.Trans != "[00:00.00]only one translation" {
		t.Errorf("trans = %q", ly.Trans)
	}
}

// TestDecodeKugouLyricLRC covers the plain base64 LRC branch, which must be
// passed through untouched apart from trimming.
func TestDecodeKugouLyricLRC(t *testing.T) {
	src := "[01:12.12]第一行\n[01:15.40]第二行\n"
	ly := DecodeKugouLyric(base64.StdEncoding.EncodeToString([]byte(src)), false)
	if ly.Format != "lrc" {
		t.Errorf("format = %q, want lrc", ly.Format)
	}
	if ly.Plain != "[01:12.12]第一行\n[01:15.40]第二行" {
		t.Errorf("plain = %q", ly.Plain)
	}
}

// TestDecodeKugouPayloadEmpty guards the malformed-input branches.
func TestDecodeKugouPayloadEmpty(t *testing.T) {
	if s, ok := decodeKugouPayload(""); ok {
		t.Errorf("empty input: got %q / %v", s, ok)
	}
	if s, ok := decodeKugouPayload("not-base64!!!"); ok {
		t.Errorf("garbage input: got %q / %v", s, ok)
	}
}

// TestKrcTagAndParseLRCLine checks the time helpers are self-consistent,
// including the 2- and 3-decimal tag formats.
func TestKrcTagAndParseLRCLine(t *testing.T) {
	for _, c := range []struct {
		ms   int
		want string
	}{
		{0, "[00:00.00]"},
		{150, "[00:00.15]"},
		{15320, "[00:15.32]"},
		{29264, "[00:29.26]"},
		{802400, "[13:22.40]"},
		{-5, "[00:00.00]"},
	} {
		if got := krcTag(c.ms); got != c.want {
			t.Errorf("krcTag(%d) = %q, want %q", c.ms, got, c.want)
		}
	}
	for _, c := range []struct {
		in     string
		wantMS int64
		text   string
	}{
		{"[13:22.40]周杰伦 晴天", 802400, "周杰伦 晴天"},
		{"[00:29.264]故事的小黄花", 29264, "故事的小黄花"},
		{"[01:02.300]三个小数", 62300, "三个小数"},
		{"没有标签的行", -1, "没有标签的行"},
		{"[ar:周杰伦]", -1, "[ar:周杰伦]"},
	} {
		ms, text := ParseLRCLine(c.in)
		if ms != c.wantMS || text != c.text {
			t.Errorf("ParseLRCLine(%q) = %d %q, want %d %q", c.in, ms, text, c.wantMS, c.text)
		}
	}
}
