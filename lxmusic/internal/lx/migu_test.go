package lx

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestMiguRegistered pins that the help text's promise keeps up with the
// registry: commands.go advertises mg, so Sources() must actually list it. It is
// listed as an audio source because listen-url hands over a 128k MP3 with no
// login - the ceiling is PQ, not the matrix the search announces.
func TestMiguRegistered(t *testing.T) {
	Register(NewClient())
	p, err := GetProvider("mg")
	if err != nil {
		t.Fatalf("mg is not registered: %v", err)
	}
	if p.ID() != "mg" || p.Name() != "咪咕音乐" {
		t.Errorf("mg identity = %q / %q", p.ID(), p.Name())
	}
	if !p.AnonymousAudio() {
		t.Error("mg denies anonymous audio; listen-url serves 128k to an unauthenticated client")
	}
	if _, err := GetProvider("MG"); err != nil {
		t.Errorf("mg lookup is case sensitive: %v", err)
	}
	ids := Sources()
	if !contains(ids, "mg") {
		t.Errorf("Sources() = %v, mg is missing", ids)
	}
}

// TestMiguFormatTypes pins the rendition vocabulary. ZQ is one spelling of the
// 24-bit tier and ZQ24 the other, and both must land on it.
func TestMiguFormatTypes(t *testing.T) {
	for fmt, want := range map[string]string{
		"PQ": Q128, "HQ": Q320, "SQ": QFlac, "ZQ": QFlac24, "ZQ24": QFlac24,
	} {
		if got := mgFormatTypes[fmt]; got != want {
			t.Errorf("mgFormatTypes[%q] = %q, want %q", fmt, got, want)
		}
	}
	for _, junk := range []string{"", "WAV", "FLAC", "p q"} {
		if _, ok := mgFormatTypes[junk]; ok {
			t.Errorf("mgFormatTypes has an entry for %q", junk)
		}
	}
}

// TestMiguTones pins both directions of the quality/toneFlag mapping, which is
// the seam where a downgrade gets reported.
func TestMiguTones(t *testing.T) {
	for q, want := range map[string]string{Q128: "PQ", Q320: "HQ", QFlac: "SQ", QFlac24: "ZQ24"} {
		if got := mgTone[q]; got != want {
			t.Errorf("mgTone[%q] = %q, want %q", q, got, want)
		}
	}
	for _, q := range []string{"", QAuto, QBestKnown, QWav, QApe, Q192} {
		if _, ok := mgTone[q]; ok {
			t.Errorf("mgTone has an entry for %q, which mg does not name", q)
		}
	}
	for tone, want := range map[string]string{
		"PQ": Q128, "HQ": Q320, "SQ": QFlac, "ZQ": QFlac24, "ZQ24": QFlac24, "": Q128, "bogus": Q128,
	} {
		if got := mgQualityForTone(tone); got != want {
			t.Errorf("mgQualityForTone(%q) = %q, want %q", tone, got, want)
		}
	}
}

// TestMiguLadderFrom pins the walk MusicURL takes when a tier comes back with no
// url: from the tier asked for down to PQ, never past it.
func TestMiguLadderFrom(t *testing.T) {
	for tone, want := range map[string][]string{
		"PQ":   {"PQ"},
		"HQ":   {"HQ", "PQ"},
		"SQ":   {"SQ", "HQ", "PQ"},
		"ZQ24": {"ZQ24", "SQ", "HQ", "PQ"},
		"":     {"PQ"},
		"LQ":   {"PQ"}, // outside the ladder falls back to the tier that always answers
	} {
		got := mgLadderFrom(tone)
		if len(got) != len(want) {
			t.Errorf("mgLadderFrom(%q) = %v, want %v", tone, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("mgLadderFrom(%q)[%d] = %q, want %q", tone, i, got[i], want[i])
			}
		}
	}
}

// TestMiguQualities checks dedup and the size fallback across the four field
// spellings. A size of 0 is a missing value, not a rendition.
func TestMiguQualities(t *testing.T) {
	qs := mgQualities([]mgFormat{
		{FormatType: "PQ", Asize: 100},
		{FormatType: "PQ", Asize: 200}, // duplicate tier, ignored
		{FormatType: "HQ", Isize: 300},
		{FormatType: "SQ", Size: 400},
		{FormatType: "ZQ24", AndroidSize: 500},
		{FormatType: "BOGUS", Asize: 999},
	})
	if len(qs) != 4 {
		t.Fatalf("mgQualities kept %d tiers, want 4: %v", len(qs), qs)
	}
	want := []struct {
		t    string
		size int64
	}{
		{Q128, 100}, {Q320, 300}, {QFlac, 400}, {QFlac24, 500},
	}
	for i, w := range want {
		if qs[i].Type != w.t || qs[i].Size != w.size {
			t.Errorf("position %d = %v, want %s/%d", i, qs[i], w.t, w.size)
		}
	}

	// No recognizable tier at all still gives a row to download at.
	if got := mgQualities([]mgFormat{}); len(got) != 1 || got[0].Type != Q128 {
		t.Errorf("empty format list = %v, want one 128k tier", got)
	}
	// ZQ and ZQ24 in the same list are one tier.
	if got := mgQualities([]mgFormat{{FormatType: "ZQ", Asize: 1}, {FormatType: "ZQ24", Asize: 2}}); len(got) != 1 {
		t.Errorf("ZQ and ZQ24 did not collapse: %v", got)
	}
}

// TestMiguSingers checks the separator matches the other sources so the table and
// the relevance scorer see one shape.
func TestMiguSingers(t *testing.T) {
	if got := mgSingers([]mgArtist{{Name: "周杰伦"}, {Name: " 林俊杰 "}}); got != "周杰伦、林俊杰" {
		t.Errorf("mgSingers = %q", got)
	}
	if got := mgSingers([]mgArtist{{Name: ""}, {Name: "  "}}); got != "" {
		t.Errorf("mgSingers on empties = %q", got)
	}
	if got := mgSingers([]mgArtist{{Name: "A &amp; B"}}); got != "A & B" {
		t.Errorf("mgSingers did not unescape: %q", got)
	}
	if got := mgSingers(nil); got != "" {
		t.Errorf("mgSingers(nil) = %q", got)
	}
}

// TestMiguSeconds pins the unit. The search feeds the field straight into
// formatPlayTime, which divides by 60, so it is seconds; the guard exists so a
// 5:32 song never reads as 5533 minutes if migu ever hands back milliseconds.
func TestMiguSeconds(t *testing.T) {
	for v, want := range map[FInt]int{
		0:       0,
		332:     332,
		3599:    3599,
		3600:    3600, // a one-hour track, in seconds
		332000:  332,  // the same 5:32 song, in milliseconds
		7200000: 7200, // a two-hour track, in milliseconds
		-5:      0,
	} {
		if got := mgSeconds(v); got != want {
			t.Errorf("mgSeconds(%d) = %d, want %d", v, got, want)
		}
	}
}

// TestMiguInterval parses listen-url's length field, which is "mm:ss" for a
// short song and "hh:mm:ss" past the hour.
func TestMiguInterval(t *testing.T) {
	for s, want := range map[string]int{
		"05:32":     332,
		"05:32.000": 332,
		"0:07":      7,
		"00:04:30":  270,
		"01:00:00":  3600,
		"00:00":     0,
		"00:00:00":  0,
		"332423":    0, // bare milliseconds, no colon
		"":          0,
		"未知":        0,
	} {
		if got := mgInterval(s); got != want {
			t.Errorf("mgInterval(%q) = %d, want %d", s, got, want)
		}
	}
}

// TestMiguSearchSong checks the id choice, the skip conditions, the relative
// cover path, and the mrcurl casing the search endpoint uses.
func TestMiguSearchSong(t *testing.T) {
	it := mgSearchItem{
		SongID: "3790007", CopyrightID: "60054701923",
		Name: "晴天", ArtistName: "周杰伦", Album: "叶惠美", AlbumID: "8592",
		Duration: 270, Img1: "/image/0/1/p.jpg", Img3: "/image/0/3/p.jpg",
		LrcURL: "https://x/lrc", MrcURLSnake: "https://x/mrc", TrcURL: "https://x/trc",
		AudioFormats: []mgFormat{{FormatType: "HQ", Asize: 9000000}},
	}
	s := mgSearchSong("mg", it)
	if s == nil {
		t.Fatal("mgSearchSong dropped a complete row")
	}
	if s.ID != "3790007" || s.MID != "60054701923" {
		t.Errorf("mgSearchSong ids = %q / %q, want song id / copyright id", s.ID, s.MID)
	}
	if s.Name != "晴天" || s.Singer != "周杰伦" {
		t.Errorf("mgSearchSong = %+v", s)
	}
	if s.Duration != 270 || s.PageURL != "http://music.migu.cn/v3/music/song/60054701923" {
		t.Errorf("duration/pageUrl = %d / %q", s.Duration, s.PageURL)
	}
	if s.Pic != mgCDN+"/image/0/3/p.jpg" {
		t.Errorf("relative cover was not joined to the cdn: %q", s.Pic)
	}
	if s.MrcURL != "https://x/mrc" {
		t.Errorf("lowercase mrcurl was not read: %q", s.MrcURL)
	}
	if len(s.Qualities) != 1 || s.Qualities[0].Type != Q320 || s.Qualities[0].Size != 9000000 {
		t.Errorf("qualities = %v", s.Qualities)
	}

	// Missing idents are not rows.
	if got := mgSearchSong("mg", mgSearchItem{Name: "晴天"}); got != nil {
		t.Errorf("row without a song id was kept: %+v", got)
	}
	if got := mgSearchSong("mg", mgSearchItem{SongID: "1"}); got != nil {
		t.Errorf("row without a title was kept: %+v", got)
	}
	// A cover falls through img3 to img2 to img1.
	if got := mgSearchSong("mg", mgSearchItem{SongID: "1", Name: "n", Img2: "/image/0/2/p.jpg"}); got.Pic != mgCDN+"/image/0/2/p.jpg" {
		t.Errorf("img2 was not used when img3 is empty: %q", got.Pic)
	}
	// An absolute cover url is left alone.
	if got := mgSearchSong("mg", mgSearchItem{
		SongID: "1", Name: "n", Img3: "https://other.example/p.jpg",
	}); got.Pic != "https://other.example/p.jpg" {
		t.Errorf("absolute cover was rewritten: %q", got.Pic)
	}
	// A row with no copyright id still resolves: the page link is just empty.
	if got := mgSearchSong("mg", mgSearchItem{SongID: "1", Name: "n"}); got.ID != "1" || got.MID != "" || got.PageURL != "" {
		t.Errorf("row without a copyright id = %+v", got)
	}
}

// TestMiguListenUnmarshal pins the listen-url payload shape. The spellings matter
// more than the values: data.url is the rendition, data.songItem is the detail
// record, and the lyric urls sit on the item rather than on data.
func TestMiguListenUnmarshal(t *testing.T) {
	body := `{
		"code": "000000", "info": "操作成功",
		"data": {
			"formatType": "PQ",
			"url": "https://freetyst.nf.migu.cn/public/x/MP3_128_16_Stero/60054701923.mp3?Tim=1",
			"songItem": {
				"songId": "3790007", "copyrightId": "60054701923",
				"songName": "晴天", "singer": "周杰伦",
				"artists": [{"name": "周杰伦"}],
				"album": "叶惠美", "albumId": "8592",
				"length": "00:04:30", "topQuality": "SQ",
				"albumImgs": [{"imgSizeType": "01", "img": "https://d.musicapp.migu.cn/c.webp"}],
				"lrcUrl": "https://d.musicapp.migu.cn/lrc",
				"mrcUrl": "https://d.musicapp.migu.cn/mrc",
				"trcUrl": ""
			}
		}
	}`
	var resp struct {
		Code FStr          `json:"code"`
		Data *mgListenData `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("fixture did not decode: %v", err)
	}
	if string(resp.Code) != mgOK || resp.Data == nil || resp.Data.SongItem == nil {
		t.Fatalf("response did not decode to the expected shape: %+v", resp)
	}
	if string(resp.Data.URL) == "" {
		t.Error("data.url was not read")
	}
	if got := mgQualityForTone(string(resp.Data.FormatType)); got != Q128 {
		t.Errorf("formatType mapped to %q, want %q", got, Q128)
	}
	it := resp.Data.SongItem
	if it.SongID != "3790007" || it.CopyrightID != "60054701923" || it.SongName != "晴天" {
		t.Errorf("songItem idents = %q / %q / %q", it.SongID, it.CopyrightID, it.SongName)
	}
	if mgInterval(string(it.Length)) != 270 {
		t.Errorf("length %q did not parse to 270s", string(it.Length))
	}
	if mgSingers(it.Artists) != "周杰伦" {
		t.Errorf("artists = %q", mgSingers(it.Artists))
	}
	if len(it.AlbumImgs) != 1 || it.AlbumImgs[0].Img == "" {
		t.Errorf("albumImgs = %+v", it.AlbumImgs)
	}
	if it.LrcURL == "" || it.MrcURL == "" || string(it.TrcURL) != "" {
		t.Errorf("lyric urls = %q / %q / %q", it.LrcURL, it.MrcURL, string(it.TrcURL))
	}
}

// TestMiguNoResource pins the id-shape hint: it fires for the no-resource answer
// and stays off a network error, which would otherwise get advice about song ids
// when the real problem is the connection.
func TestMiguIDError(t *testing.T) {
	if got := mgIDError("1", nil); got != nil {
		t.Errorf("mgIDError(nil) = %v", got)
	}
	wrapped := mgIDError("60054701923", errMiguNoResource)
	if !strings.Contains(wrapped.Error(), mgIDNote) {
		t.Errorf("the no-resource answer lost its hint: %v", wrapped)
	}
	if !strings.Contains(wrapped.Error(), "60054701923") {
		t.Errorf("the failing id was not echoed: %v", wrapped)
	}
	net := mgIDError("1", context.DeadlineExceeded)
	if net == nil || strings.Contains(net.Error(), mgIDNote) {
		t.Errorf("a network error was given the id hint: %v", net)
	}
}

// TestMiguIDRequired pins that a reference without a song id fails before making
// a request: the gateway's error for a bad id is indistinguishable from "no such
// song", so the empty case has to be caught locally.
func TestMiguIDRequired(t *testing.T) {
	m := &Migu{}
	if err := m.Detail(context.Background(), &Song{Source: "mg"}); err == nil ||
		!strings.Contains(err.Error(), "no song id") {
		t.Errorf("Detail with no song id = %v, want a no-song-id error", err)
	}
	if _, err := m.MusicURL(context.Background(), &Song{Source: "mg"}, Quality{Type: Q128}); err == nil ||
		!strings.Contains(err.Error(), "no song id") {
		t.Errorf("MusicURL with no song id = %v, want a no-song-id error", err)
	}
}

// TestDecryptMiguLyricShort pins the passthrough: anything under 32 chars is
// returned as-is, which is how migu marks an unencrypted payload.
func TestDecryptMiguLyricShort(t *testing.T) {
	for _, s := range []string{"", "abc", "0123456789abcdef", strings.Repeat("0", 31)} {
		if got := DecryptMiguLyric(s); got != s {
			t.Errorf("DecryptMiguLyric(%q) = %q, want unchanged", s, got)
		}
	}
}

// TestDecryptMiguLyricPassthroughNonHex: a plain lrc body served from an mrc slot
// is not hex, so it must come back untouched rather than turned into garbage.
func TestDecryptMiguLyricPassthroughNonHex(t *testing.T) {
	lrc := "[00:01.234]晴 天\n[00:03.000]周杰伦的歌"
	if got := DecryptMiguLyric(lrc); got != lrc {
		t.Errorf("DecryptMiguLyric mangled a plain lrc body: %q", got)
	}
}

// TestDecryptMiguLyricShape checks the codec's output shape: one 16-char hex word
// is eight bytes, four UTF-16 code units, so the output has exactly four runes
// per word and is always valid UTF-8.
func TestDecryptMiguLyricShape(t *testing.T) {
	bodies := map[string]int{
		strings.Repeat("0123456789abcdef", 2): 8,
		strings.Repeat("0123456789abcdef", 3): 12,
		strings.Repeat("fedcba9876543210", 4): 16,
	}
	for body, wantRunes := range bodies {
		got := DecryptMiguLyric(body)
		if utf8.RuneCountInString(got) != wantRunes {
			t.Errorf("DecryptMiguLyric produced %d runes, want %d for %d hex chars",
				utf8.RuneCountInString(got), wantRunes, len(body))
		}
	}
	// The decoder is deterministic and not constant.
	a := DecryptMiguLyric(strings.Repeat("0123456789abcdef", 4))
	b := DecryptMiguLyric(strings.Repeat("fedcba9876543210", 4))
	if a != DecryptMiguLyric(strings.Repeat("0123456789abcdef", 4)) {
		t.Error("DecryptMiguLyric is not deterministic")
	}
	if a == b {
		t.Error("DecryptMiguLyric gives the same answer for different ciphertext")
	}
}

// TestParseMiguMRC is the mrc parser: line heads become LRC tags, per-word
// markers become relative offsets, and Plain keeps the lyric without them.
func TestParseMiguMRC(t *testing.T) {
	src := "[1234,1100](1234,300)晴(1534,400)天(1934,400)\n[3000,2000](3000,1000)周(4000,1000)杰伦"
	l := parseMiguMRC(src)
	if l == nil {
		t.Fatal("parseMiguMRC returned nil")
	}
	if l.Plain != "[00:01.234]晴天\n[00:03.000]周杰伦" {
		t.Errorf("Plain = %q", l.Plain)
	}
	if l.Raw != "[00:01.234]<0,300><300,400>晴<700,400>天\n[00:03.000]<0,1000><1000,1000>周" {
		t.Errorf("Raw = %q", l.Raw)
	}
	// The word marker is a prefix, not a suffix: "(1234,300)晴" is the word 晴
	// starting at 1234 and lasting 300 ms. Kuwo's "<0,0>" prefix uses the same
	// convention, and both are what the karaoke view renders against.
	if l.Format != "lrc" {
		t.Errorf("Format = %q", l.Format)
	}

	// No line heads, no lyric.
	if got := parseMiguMRC("nothing but text\nwithout timings"); got != nil {
		t.Errorf("parseMiguMRC on plain text = %+v, want nil", got)
	}
	// Lines without per-word markers still produce a plain lyric.
	if got := parseMiguMRC("[1000,500]一句没有词标记的歌词"); got == nil ||
		!strings.Contains(got.Plain, "[00:01.000]") || !strings.Contains(got.Plain, "一句") {
		t.Errorf("parseMiguMRC without word markers = %+v", got)
	}
	// Short lines are skipped, which is the app's line.length < 6 rule.
	if got := parseMiguMRC("[1]x"); got != nil {
		t.Errorf("parseMiguMRC kept a too-short line: %+v", got)
	}
}

// TestMiguTag covers the timestamp rendering.
func TestMiguTag(t *testing.T) {
	for ms, want := range map[int]string{
		0:         "[00:00.000]",
		1234:      "[00:01.234]",
		3000:      "[00:03.000]",
		123456789: "[2057:36.789]",
	} {
		if got := mgMrcTag(ms); got != want {
			t.Errorf("mgMrcTag(%d) = %q, want %q", ms, got, want)
		}
	}
}
