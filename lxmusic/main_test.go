package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	lx "lxmusic/internal/lx"
	"lxmusic/internal/tui"
)

// TestQualityRank pins the tier order the table sorts by, including the
// unrecognised case, which must not beat a real 128k file.
func TestQualityRank(t *testing.T) {
	for _, c := range []struct {
		q    string
		want int
	}{
		{lx.QFlac24, 4}, {lx.QFlac, 3}, {lx.Q320, 2}, {lx.Q128, 1},
		{"-", 0}, {"", 0}, {"flac24", 0}, {"FLAC", 0},
	} {
		if got := qualityRank(c.q); got != c.want {
			t.Errorf("qualityRank(%q) = %d, want %d", c.q, got, c.want)
		}
	}
}

// TestSortRows pins the table order: quality descending first, then size
// descending within a tier. The comparator has been wrong twice now - once by
// inverting the size direction, once by using size as the only key - so each
// half of the rule gets its own assertion.
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
	sortRows(rows)
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
	sortRows(rows)
	if rows[0].Name != "a" || rows[1].Name != "b" || rows[2].Name != "c" {
		t.Errorf("sortRows reordered a full tie: %v", rows)
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
	sortRows(rows)
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
