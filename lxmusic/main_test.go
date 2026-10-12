package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	lx "lxmusic/internal/lx"
	"lxmusic/internal/tui"
)

// TestQualityRank pins the tier order the table sorts by. The ranks are the
// positions in lx.qualityOrder, so this also pins the ordering the desktop app
// itself uses — flac24bit, flac, wav, ape, 320k, 192k, 128k — which is what
// keeps a user source that only announces "wav" from sinking to the bottom of
// the table as an unknown tier. Unrecognised labels must not beat a real 128k.
func TestQualityRank(t *testing.T) {
	for _, c := range []struct {
		q    string
		want int
	}{
		{lx.QFlac24, 7}, {lx.QFlac, 6}, {lx.QWav, 5}, {lx.QApe, 4},
		{lx.Q320, 3}, {lx.Q192, 2}, {lx.Q128, 1},
		{"-", 0}, {"", 0}, {"flac24", 0}, {"FLAC", 0}, {"wav24", 0},
	} {
		if got := qualityRank(c.q); got != c.want {
			t.Errorf("qualityRank(%q) = %d, want %d", c.q, got, c.want)
		}
	}
	// The order must be strictly descending, whatever the vocabulary grows to.
	ranks := []int{}
	for _, q := range lx.QualityList() {
		ranks = append(ranks, lx.QualityRank(q))
	}
	for i := 1; i < len(ranks); i++ {
		if ranks[i] >= ranks[i-1] {
			t.Fatalf("qualityOrder is not strictly descending at %d: %v", i, ranks)
		}
	}
}

// TestSortRows pins the table order: relevance descending, then quality
// descending, then size descending within a tier. The comparator has been wrong
// three times now - once by inverting the size direction, once by using size as
// the only key, once by sorting the rows while a parallel score slice stayed
// behind - so each half of the rule gets its own assertion.
func TestSortRows(t *testing.T) {
	rows := []tui.Row{
		{Name: "unknown"},
		{Name: "tiny320", Quality: lx.Q320, Size: 1024},
		{Name: "big320", Quality: lx.Q320, Size: 90000000},
		{Name: "noSizeFlac", Quality: lx.QFlac},
		{Name: "flac24", Quality: lx.QFlac24, Size: 60000000},
		{Name: "small128", Quality: lx.Q128, Size: 2000000},
		{Name: "bigFlac", Quality: lx.QFlac, Size: 52000000},
		{Name: "unknownNoSize"},
	}
	// No keyword: every row scores the same relevance, so quality and size take
	// over and the ordering is the plain fidelity order.
	sortRows(rows, "")
	want := []string{"flac24", "bigFlac", "noSizeFlac", "big320", "tiny320", "small128", "unknown", "unknownNoSize"}
	if len(rows) != len(want) {
		t.Fatalf("sortRows returned %d rows, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if r.Name != want[i] {
			t.Errorf("position %d = %q, want %q", i, r.Name, want[i])
		}
	}
	// Size must still descend inside a quality tier.
	if rows[3].Size <= rows[4].Size {
		t.Errorf("320k tier not descending by size: %d then %d", rows[3].Size, rows[4].Size)
	}
}

// TestSortRowsStableOnTies checks a full tie keeps the platforms' own order,
// rather than reordering rows the user had no reason to expect to move.
func TestSortRowsStableOnTies(t *testing.T) {
	rows := []tui.Row{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	sortRows(rows, "")
	if rows[0].Name != "a" || rows[1].Name != "b" || rows[2].Name != "c" {
		t.Errorf("sortRows reordered a full tie: %v", rows)
	}
}

// TestSortRowsRelevanceFirst is the regression test for the README's own
// complaint: a cover recorded in lossless used to rank above the original that
// only exists at 320k. Relevance now leads the sort, so the original comes
// first no matter what the bitrates are.
func TestSortRowsRelevanceFirst(t *testing.T) {
	rows := []tui.Row{
		{Name: "晴天", Artist: "林俊杰", Quality: lx.QFlac24, Size: 60000000}, // cover, lossless
		{Name: "晴天", Artist: "周杰伦", Quality: lx.Q320, Size: 9000000},     // the original, 320k
		{Name: "晴天", Artist: "周杰伦", Quality: lx.Q128, Size: 4000000},     // the original, 128k
		{Name: "七里香", Artist: "周杰伦", Quality: lx.QFlac, Size: 50000000},  // unrelated song
	}
	sortRows(rows, "晴天 周杰伦")
	want := []string{"周杰伦", "周杰伦", "林俊杰", "周杰伦"}
	for i, r := range rows {
		if r.Artist != want[i] {
			t.Errorf("position %d = %q - %q, want artist %q", i, r.Name, r.Artist, want[i])
		}
	}
	// Within the two matching rows quality still orders them.
	if rows[0].Quality != lx.Q320 || rows[1].Quality != lx.Q128 {
		t.Errorf("quality did not break the tie within a relevance group: %v then %v",
			rows[0].Quality, rows[1].Quality)
	}
}

// TestSortRowsVersionTiebreak checks that an alternate recording does not outrank
// the plain one when both match the keyword.
func TestSortRowsVersionTiebreak(t *testing.T) {
	rows := []tui.Row{
		{Name: "晴天 (现场版)", Artist: "周杰伦", Quality: lx.Q128, Size: 1},
		{Name: "晴天", Artist: "周杰伦", Quality: lx.Q128, Size: 1},
	}
	sortRows(rows, "晴天")
	if rows[0].Name != "晴天" {
		t.Errorf("version mismatch was not demoted: first row = %q", rows[0].Name)
	}
}

// TestCapAfterSort pins that the rows that survive the cap are the best, not
// the first fetched. Capping before the sort would show an arbitrary page.
func TestCapAfterSort(t *testing.T) {
	// Built worst-first, so a cap that ran before the sort would have dropped
	// the flac24bit row that ends up on top.
	rows := make([]tui.Row, 0, maxResults+10)
	for i := 0; i < maxResults+10; i++ {
		rows = append(rows, tui.Row{
			Name:    "row",
			Quality: lx.Q128,
			Size:    int64(i),
		})
	}
	rows = append(rows, tui.Row{Name: "best", Quality: lx.QFlac24, Size: 1})
	sortRows(rows, "")
	if len(rows) > maxResults {
		rows = rows[:maxResults]
	}
	if len(rows) != maxResults {
		t.Fatalf("capped to %d rows, want %d", len(rows), maxResults)
	}
	if rows[0].Name != "best" {
		t.Errorf("first row = %q, want the flac24bit row to sort above 128k", rows[0].Name)
	}
	// Sizes descend within each quality tier; they are not required to descend
	// across tiers, since quality is the primary key.
	for i := 1; i < len(rows); i++ {
		if rows[i].Quality != rows[i-1].Quality {
			continue
		}
		if rows[i].Size > rows[i-1].Size {
			t.Fatalf("not descending at position %d: %d after %d", i, rows[i].Size, rows[i-1].Size)
		}
	}
}

// TestToRowsCarriesSize checks the new column comes from the best rendition -
// the one --quality auto would download - not from an arbitrary quality.
func TestToRowsCarriesSize(t *testing.T) {
	res := &lx.SearchResult{Source: "kw", List: []*lx.Song{
		{Name: "a", Qualities: []lx.Quality{{Type: lx.Q128, Size: 1024}, {Type: lx.QFlac, Size: 50000000}}},
		{Name: "b", Qualities: []lx.Quality{{Type: lx.QFlac}}},
	}}
	out := toRows(res)
	if len(out) != 2 {
		t.Fatalf("toRows returned %d rows, want 2", len(out))
	}
	if out[0].Size != 50000000 {
		t.Errorf("row 1 size = %d, want the flac size", out[0].Size)
	}
	if out[1].Size != 0 {
		t.Errorf("row 2 size = %d, want 0 (none announced)", out[1].Size)
	}
	if out[0].Index != 0 || out[1].Index != 0 {
		t.Errorf("toRows assigned indexes (%d, %d); cmdSearch numbers after sorting",
			out[0].Index, out[1].Index)
	}
}

// TestSavePathFor pins that the confirm prompt's destination is computed from
// the row alone. It must not touch the network: it runs between the Enter
// keypress and the prompt appearing, so a round trip in here freezes the UI.
func TestSavePathFor(t *testing.T) {
	outDir := filepath.Join(os.TempDir(), "lxmusic-out")
	cases := []struct {
		row  tui.Row
		want string
	}{
		{tui.Row{Name: "晴天", Quality: lx.QFlac}, filepath.Join(outDir, "晴天.flac")},
		{tui.Row{Name: "晴天", Quality: lx.QFlac24}, filepath.Join(outDir, "晴天.flac")},
		{tui.Row{Name: "晴天", Quality: lx.Q320}, filepath.Join(outDir, "晴天.mp3")},
		{tui.Row{Name: "晴天", Quality: "-"}, filepath.Join(outDir, "晴天.mp3")},
		// An unnamed row falls back to its reference. The sanitizer rewrites the
		// colon, which keeps the path legal on Windows.
		{tui.Row{Name: "", Ref: "kw:228908"}, filepath.Join(outDir, "kw_228908.mp3")},
	}
	for _, c := range cases {
		if got := savePathFor(c.row, outDir); got != c.want {
			t.Errorf("savePathFor(%+v) = %q, want %q", c.row, got, c.want)
		}
	}
	// Whatever it returns must be inside outDir: the prompt promises a path.
	if got := savePathFor(tui.Row{Name: strings.Repeat("x", 4000), Quality: lx.Q128}, outDir); !strings.HasPrefix(got, outDir) {
		t.Errorf("savePathFor escaped outDir: %q", got)
	}
}

// TestDownloadDir pins that downloads default into a music subdirectory of the
// directory the command was run from, whatever that directory happens to be.
// The assertion is about the shape, not the absolute path, so it holds no
// matter where the test process lives.
func TestDownloadDir(t *testing.T) {
	wantBase, wantParent := "music", ""
	if wd, err := os.Getwd(); err == nil {
		wantParent = wd
	}
	got := downloadDir()
	if got == "" || got == "." {
		t.Fatalf("downloadDir = %q, want a music directory", got)
	}
	if base := filepath.Base(got); base != wantBase {
		t.Errorf("downloadDir base = %q, want %q", base, wantBase)
	}
	if wantParent != "" {
		if parent := filepath.Dir(got); parent != wantParent {
			t.Errorf("downloadDir parent = %q, want the working directory %q", parent, wantParent)
		}
	}
	// It must be usable as an --out value: MkdirAll has to be able to create it.
	dir := filepath.Join(os.TempDir(), "lxmusic-dir-test", "music")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("the default download directory is not creatable: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(dir))
}

// TestPumpProgress pins the stop handshake: stop must return only after the
// final line has been written, and it must actually end the goroutine rather
// than leaving it writing forever after the download finished. A second stop
// call must be harmless - the caller sits in a control flow a future edit could
// wrap in a defer.
func TestPumpProgress(t *testing.T) {
	old := progressTick
	progressTick = 5 * time.Millisecond
	defer func() { progressTick = old }()

	var buf bytes.Buffer
	p := &tui.DownloadProgress{}
	stop := pumpProgress(&buf, p, "kw:1")

	time.Sleep(20 * time.Millisecond) // a few frames with nothing to report yet
	p.Report(1000*1000, 2*1000*1000)
	time.Sleep(20 * time.Millisecond)
	stop()
	out := buf.String()

	if !strings.Contains(out, "\r") {
		t.Errorf("pump emitted %q, want redraw frames separated by \\r", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("pump emitted %q, want the progress bar", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("pump output does not end with a newline, so the saved line would share it: %q", out)
	}
	n := len(out)
	time.Sleep(20 * time.Millisecond)
	if got := buf.String(); len(got) != n {
		t.Errorf("pump kept writing after stop: %d more bytes", len(got)-n)
	}
	// A duplicate stop must not panic or block; the first one already finished.
	done := make(chan struct{})
	go func() { stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("a second stop call blocked")
	}
}

// TestPreviewNote pins the size test that decides a fragment from a track. The
// 181521 byte figure is kuwo's real anonymous preview of a 332 s song, and it is
// the case that used to be reported as a successful save.
func TestPreviewNote(t *testing.T) {
	for _, c := range []struct {
		name     string
		duration int
		got      int64
		wantHit  bool
	}{
		{"kuwo preview of a 5:32 song", 332, 181521, true},
		{"a real 5:32 mp3", 332, 3984000, false},
		{"exactly the floor is a track", 332, 332 * 16000 * 3 / 4, false},
		{"one byte under the floor", 332, 332*16000*3/4 - 1, true},
		{"no runtime, small file", 0, 181521, true},
		{"no runtime, big file", 0, 500 * 1024, false},
		{"nothing downloaded", 332, 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := previewNote(c.duration, c.got)
			if (got != "") != c.wantHit {
				t.Fatalf("previewNote(%d, %d) = %q, want %v hit", c.duration, c.got, got, c.wantHit)
			}
			if c.wantHit && c.duration > 0 && !strings.Contains(got, "05:32") {
				t.Errorf("warning omits the runtime: %q", got)
			}
		})
	}
}

// noLyric is the smallest provider there is: tagDownload only ever calls Lyric
// on it, and it answers with nothing — the ordinary case for a song a platform
// will not give out lyrics for.
type noLyric struct{}

func (noLyric) ID() string                                                         { return "fake" }
func (noLyric) Name() string                                                       { return "fake" }
func (noLyric) Search(context.Context, string, int, int) (*lx.SearchResult, error) { return nil, nil }
func (noLyric) MusicURL(context.Context, *lx.Song, lx.Quality) (*lx.PlayerURL, error) {
	return nil, nil
}
func (noLyric) Lyric(context.Context, *lx.Song) (*lx.Lyric, error) {
	return nil, errors.New("no lyric")
}
func (noLyric) Pic(context.Context, *lx.Song) (string, error) { return "", nil }
func (noLyric) Detail(context.Context, *lx.Song) error        { return nil }
func (noLyric) AnonymousAudio() bool                          { return false }

// TestTagDownloadSkips pins the safety half: a format this writer does not know,
// and a song with no name, are both left byte for byte alone. Neither path may
// touch the network, so both the client and the provider are nil — the guards
// must come before either is reached.
func TestTagDownloadSkips(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	for _, ext := range []string{".ape", ".ogg", ".m4a", ".wav", ".wma"} {
		path := filepath.Join(dir, "x"+ext)
		writeMain(t, path, []byte("raw audio"))
		note, err := tagDownload(ctx, nil, nil, &lx.Song{Name: "晴天"}, path)
		if err != nil || note != "" {
			t.Errorf("%s: (%q, %v) - an untaggable format must be left alone", ext, note, err)
		}
		if b, _ := os.ReadFile(path); !bytes.Equal(b, []byte("raw audio")) {
			t.Errorf("%s was rewritten", ext)
		}
	}
	for _, name := range []string{"", "   "} {
		path := filepath.Join(dir, "nameless.mp3")
		writeMain(t, path, []byte("raw audio"))
		note, err := tagDownload(ctx, nil, nil, &lx.Song{Name: name}, path)
		if err != nil || note != "" {
			t.Errorf("name %q: (%q, %v) - nothing to write must not write", name, note, err)
		}
		if b, _ := os.ReadFile(path); !bytes.Equal(b, []byte("raw audio")) {
			t.Error("a song with no name rewrote its file")
		}
	}
}

// TestTagDownloadWrites is the happy path with no network in it: no cover url, so
// the client is never touched, and no lyric, so noLyric stands in for the source.
// The note must say exactly what landed, and the audio must survive the write.
func TestTagDownloadWrites(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	audio := []byte{0xff, 0xfb, 0x90, 0x64}

	path := filepath.Join(dir, "晴天.mp3")
	writeMain(t, path, audio)
	note, err := tagDownload(ctx, nil, noLyric{},
		&lx.Song{Name: "晴天", Singer: "周杰伦", Album: "叶惠美"}, path)
	if err != nil {
		t.Fatal(err)
	}
	if note != "已写入标签 信息" {
		t.Errorf("note = %q, want 已写入标签 信息", note)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("ID3")) {
		t.Error("no ID3 tag was written")
	}
	if !bytes.HasSuffix(got, audio) {
		t.Error("the audio was not preserved")
	}

	// An uppercase extension is tagged the same way.
	path2 := filepath.Join(dir, "UPPER.MP3")
	writeMain(t, path2, audio)
	note, err = tagDownload(ctx, nil, noLyric{}, &lx.Song{Name: "晴天"}, path2)
	if err != nil || note == "" {
		t.Errorf(".MP3: (%q, %v)", note, err)
	}

	// Re-running on an already tagged file must not stack a second tag.
	note, err = tagDownload(ctx, nil, noLyric{},
		&lx.Song{Name: "晴天", Singer: "周杰伦", Album: "叶惠美"}, path)
	if err != nil {
		t.Fatal(err)
	}
	if note != "已写入标签 信息" {
		t.Errorf("re-tag note = %q, want 已写入标签 信息", note)
	}
	got, _ = os.ReadFile(path)
	// The old tag is stripped, so only one ID3 marker may remain.
	if n := bytes.Count(got, []byte("ID3")); n != 1 {
		t.Errorf("the re-tag left %d ID3 markers, want 1", n)
	}
	if !bytes.HasSuffix(got, audio) {
		t.Error("the re-tag did not preserve the audio")
	}
}

func writeMain(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
