package lx

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestAudioExtURLWins pins that the extension comes from the served bytes, not
// from the quality that was requested. A source can hand back a different format
// than the label, and naming the file after the request would lie about it.
func TestAudioExtURLWins(t *testing.T) {
	// A real kuwo flac rendition: the query is "$"-delimited and carries ".apk"
	// inside a spoofed User-Agent, so only the path may be inspected.
	flac := "http://car-er.kuwo.cn/x/resource/130/trackmedia/F000000B3Ekk1I79hc_1.flac" +
		"?bitrate$2000&format$flac&source$kwplayercar_ar_6.0.0.9_B_jiakong_vh.apk&type$convert_url_with_sign"
	if got := AudioExt(Q320, flac); got != ".flac" {
		t.Errorf("AudioExt(%q, flac url) = %q, want .flac", Q320, got)
	}
	if got := AudioExt(QFlac, "http://s/t/trackmedia/M800000B3Ekk1I79hc.mp3?vkey=x"); got != ".mp3" {
		t.Errorf("AudioExt(%q, mp3 url) = %q, want .mp3", QFlac, got)
	}
	if got := AudioExt(QFlac, "http://s/c.mp3?v=1.mp4"); got != ".mp3" {
		t.Errorf("query extension leaked into the result: %q", got)
	}
	if got := AudioExt(QFlac, "http://s/c.flac#frag.mp3"); got != ".flac" {
		t.Errorf("fragment leaked into the result: %q", got)
	}
	// A dot in an intermediate segment must not be mistaken for the extension.
	if got := AudioExt(QFlac, "http://s/a.b/c.mp3"); got != ".mp3" {
		t.Errorf("directory dot mistaken for extension: %q", got)
	}
}

// TestAudioExtQualityFallback covers streams with no extension in the path, which
// is what most gateway URLs look like.
func TestAudioExtQualityFallback(t *testing.T) {
	for _, c := range []struct {
		q    string
		want string
	}{
		{Q128, ".mp3"},
		{Q192, ".mp3"},
		{Q320, ".mp3"},
		{QFlac, ".flac"},
		{QFlac24, ".flac"},
		{QWav, ".wav"},
		{QApe, ".ape"},
		{"", ".mp3"},
		{"auto", ".mp3"},
	} {
		if got := AudioExt(c.q, "http://stream.qqmusic.qq.com/fid?vkey=abc"); got != c.want {
			t.Errorf("AudioExt(%q, no-ext url) = %q, want %q", c.q, got, c.want)
		}
	}
}

// TestLocalPath guards the Windows case: url.Parse reads "C:\foo" as scheme "c",
// so without asking filepath.IsAbs first every absolute local path would be sent
// to the HTTP client as a URL.
func TestLocalPath(t *testing.T) {
	type pair struct {
		in   string
		want bool
	}
	checks := []pair{
		{"http://example.com/a.js", false},
		{"https://example.com/a.js", false},
		{"file:///tmp/a.js", true},
		{"scripts/a.js", true},
		{"a.js", true},
	}
	if runtime.GOOS == "windows" {
		// The point of the fix: these parse as scheme "c", not "".
		for _, p := range []string{`C:\Users\jmrx\Downloads\a.js`, "C:/Users/jmrx/a.js", `\\server\share\a.js`} {
			if !filepath.IsAbs(p) {
				t.Fatalf("test premise broken: %q is not absolute on %s", p, runtime.GOOS)
			}
			checks = append(checks, pair{p, true})
		}
	}
	for _, c := range checks {
		if got := localPath(c.in); got != c.want {
			t.Errorf("localPath(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestBestOf pins the fidelity order used when nothing else says what the best
// tier is - a song known only by id, where the script's own capability table is
// the only source of truth. Order matters both ways: the highest wins, and
// unrecognised names never beat a real tier.
func TestBestOf(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{[]string{}, ""},
		{nil, ""},
		{[]string{Q128, Q320, QFlac, QFlac24}, QFlac24},
		{[]string{QFlac24}, QFlac24},
		{[]string{Q128}, Q128},
		// Listing order must not matter.
		{[]string{Q128, Q320, QFlac24, QFlac}, QFlac24},
		{[]string{QFlac, Q128, Q320}, QFlac},
		{[]string{Q320, Q128}, Q320},
		// Duplicates are harmless.
		{[]string{QFlac, QFlac}, QFlac},
		// Unknown names are ignored, not trusted. ape and wav are real tiers, so
		// only names outside the desktop app's vocabulary are dropped.
		{[]string{QApe}, QApe},
		{[]string{QWav, Q128}, QWav},
		{[]string{"ogg", "auto"}, ""},
		{[]string{"ogg", Q128}, Q128},
		// Case is not folded: the tiers are exact strings.
		{[]string{"FLAC"}, ""},
	} {
		if got := bestOf(c.in); got != c.want {
			t.Errorf("bestOf(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestQualityByType pins the fallback. The load-bearing case is the last one: a
// request naming a tier the matrix does not list must be passed through, not
// replaced by the first known one. mg announces a song known only by id with PQ
// alone, so substituting turned an explicit flac24bit request into 128k with no
// note at all - the user asked for one thing and got another, silently.
func TestQualityByType(t *testing.T) {
	matrix := []Quality{{Type: Q128, Size: 1024}, {Type: Q320, Size: 2048}, {Type: QFlac, Size: 52428800}}
	for _, c := range []struct {
		name string
		q    []Quality
		req  string
		want Quality
	}{
		{"exact match keeps its size", matrix, Q320, Quality{Type: Q320, Size: 2048}},
		{"auto settles on the best", matrix, QAuto, Quality{Type: QFlac, Size: 52428800}},
		{"best settles on the best", matrix, QBestKnown, Quality{Type: QFlac, Size: 52428800}},
		{"empty matrix passes the request through", nil, QFlac24, Quality{Type: QFlac24}},
		{"auto with no matrix stays auto", nil, QAuto, Quality{Type: QAuto}},
		{"tier the matrix lacks is passed through", matrix, QFlac24, Quality{Type: QFlac24}},
		{"tier the matrix lacks is not the first entry", matrix, QWav, Quality{Type: QWav}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := &Song{Qualities: c.q}
			if got := s.QualityByType(c.req); got != c.want {
				t.Errorf("QualityByType(%q) = %+v, want %+v", c.req, got, c.want)
			}
		})
	}
	// The regression as it actually presented: mg detail announces PQ alone for a
	// song known only by id, and the request is for the top tier.
	if got := (&Song{Qualities: []Quality{{Type: Q128}}}).QualityByType(QFlac24); got.Type != QFlac24 {
		t.Errorf("mg-style matrix substituted %q for a flac24bit request", got.Type)
	}
}
