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

// TestApplyServed pins what the table shows after a rendition is probed. The
// load-bearing case is the migu one at the end: the catalogue advertises
// flac24bit at 52.6MB, the gateway serves a 128k mp3, and the row has to come
// out reading 128k with the 128k size - never 128k next to the flac number,
// which is what the probe used to leave on screen.
func TestApplyServed(t *testing.T) {
	matrix := func() *lx.Song {
		return &lx.Song{Source: "mg", Qualities: []lx.Quality{
			{Type: lx.Q128, Size: 4018053},
			{Type: lx.Q320, Size: 10044816},
			{Type: lx.QFlac, Size: 29892504},
			{Type: lx.QFlac24, Size: 55158550},
		}}
	}
	row := func(q string, song *lx.Song) tui.Row {
		return tui.Row{Quality: q, Size: 55158550, Song: song}
	}
	for _, c := range []struct {
		name        string
		q           string
		song        *lx.Song
		pl          *lx.PlayerURL
		wantQuality string
		wantSize    int64
		wantChanged bool
		wantDown    bool
	}{
		{
			name:        "no song leaves the row alone",
			q:           lx.QFlac24,
			song:        nil,
			pl:          &lx.PlayerURL{Quality: lx.Q128},
			wantQuality: lx.QFlac24,
			wantSize:    55158550,
		},
		{
			name:        "a failed probe leaves the row alone",
			q:           lx.QFlac24,
			song:        matrix(),
			pl:          nil,
			wantQuality: lx.QFlac24,
			wantSize:    55158550,
		},
		{
			name:        "an empty answer leaves the row alone",
			q:           lx.QFlac24,
			song:        matrix(),
			pl:          &lx.PlayerURL{Quality: ""},
			wantQuality: lx.QFlac24,
			wantSize:    55158550,
		},
		{
			name:        "the same tier is not a change",
			q:           lx.QFlac24,
			song:        matrix(),
			pl:          &lx.PlayerURL{Quality: lx.QFlac24},
			wantQuality: lx.QFlac24,
			wantSize:    55158550,
		},
		{
			name:        "the size follows the served tier",
			q:           lx.QFlac24,
			song:        matrix(),
			pl:          &lx.PlayerURL{Quality: lx.Q128},
			wantQuality: lx.Q128,
			wantSize:    4018053,
			wantChanged: true,
			wantDown:    true,
		},
		{
			name:        "an unsized tier falls back to the url size",
			q:           lx.QFlac24,
			song:        &lx.Song{Qualities: []lx.Quality{{Type: lx.QFlac24, Size: 55158550}}},
			pl:          &lx.PlayerURL{Quality: lx.Q128, Size: 3900000},
			wantQuality: lx.Q128,
			wantSize:    3900000,
			wantChanged: true,
			wantDown:    true,
		},
		{
			name:        "no size anywhere reads unknown, not the old number",
			q:           lx.QFlac24,
			song:        &lx.Song{Qualities: []lx.Quality{{Type: lx.QFlac24, Size: 55158550}}},
			pl:          &lx.PlayerURL{Quality: lx.Q128},
			wantQuality: lx.Q128,
			wantSize:    0,
			wantChanged: true,
			wantDown:    true,
		},
		{
			// The probe normally confirms or lowers the tier, but a row built from
			// a bare reference advertises nothing at all, so a real tier is a gain.
			name:        "a higher served tier is a change, not a downgrade",
			q:           lx.Q320,
			song:        matrix(),
			pl:          &lx.PlayerURL{Quality: lx.QFlac24},
			wantQuality: lx.QFlac24,
			wantSize:    55158550,
			wantChanged: true,
		},
		{
			name:        "an unknown served tier is not treated as the best",
			q:           lx.QFlac24,
			song:        matrix(),
			pl:          &lx.PlayerURL{Quality: "-"},
			wantQuality: "-",
			wantSize:    0,
			wantChanged: true,
			wantDown:    true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := row(c.q, c.song)
			changed, down := applyServed(&r, c.pl)
			if r.Quality != c.wantQuality || r.Size != c.wantSize {
				t.Errorf("row = %s/%d, want %s/%d", r.Quality, r.Size, c.wantQuality, c.wantSize)
			}
			if changed != c.wantChanged || down != c.wantDown {
				t.Errorf("(changed, down) = (%v, %v), want (%v, %v)", changed, down, c.wantChanged, c.wantDown)
			}
		})
	}
	// A row built from a bare reference has neither a song nor a size, and must
	// not panic or change.
	var bare tui.Row
	if changed, down := applyServed(&bare, &lx.PlayerURL{Quality: lx.Q128}); changed || down {
		t.Errorf("a bare row must not change: (%v, %v)", changed, down)
	}
}

// slowProvider never answers: it sits on its context until the caller cancels.
// That is the shape of a gateway that is down, which is what probeBudget exists
// to bound.
type slowProvider struct{}

func (slowProvider) ID() string   { return "slow" }
func (slowProvider) Name() string { return "slow" }
func (slowProvider) Search(context.Context, string, int, int) (*lx.SearchResult, error) {
	return nil, nil
}
func (slowProvider) MusicURL(ctx context.Context, s *lx.Song, q lx.Quality) (*lx.PlayerURL, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (slowProvider) Lyric(context.Context, *lx.Song) (*lx.Lyric, error) {
	return nil, errors.New("no lyric")
}
func (slowProvider) Pic(context.Context, *lx.Song) (string, error) { return "", nil }
func (slowProvider) Detail(context.Context, *lx.Song) error        { return nil }
func (slowProvider) AnonymousAudio() bool                          { return true }

// answerProvider answers with one fixed tier, whatever was asked.
type answerProvider struct{ pl *lx.PlayerURL }

func (answerProvider) ID() string   { return "ans" }
func (answerProvider) Name() string { return "ans" }
func (answerProvider) Search(context.Context, string, int, int) (*lx.SearchResult, error) {
	return nil, nil
}
func (a answerProvider) MusicURL(context.Context, *lx.Song, lx.Quality) (*lx.PlayerURL, error) {
	return a.pl, nil
}
func (answerProvider) Lyric(context.Context, *lx.Song) (*lx.Lyric, error) {
	return nil, errors.New("no lyric")
}
func (answerProvider) Pic(context.Context, *lx.Song) (string, error) { return "", nil }
func (answerProvider) Detail(context.Context, *lx.Song) error        { return nil }
func (answerProvider) AnonymousAudio() bool                          { return true }

func stubProviders(t *testing.T, p lx.Provider) {
	t.Helper()
	old := providerFor
	providerFor = func(id string) (lx.Provider, error) { return p, nil }
	t.Cleanup(func() { providerFor = old })
}

// TestServedRowsBudget pins that the rendition check cannot run past its own
// budget. It once borrowed the search's 120 s deadline and exhausted it, so the
// sort ran on rows nobody had measured and the table still advertised the tier
// the download would not give. Here every row hangs on its context, so the only
// thing that can end the call is the budget.
func TestServedRowsBudget(t *testing.T) {
	oldBudget := probeBudget
	probeBudget = 200 * time.Millisecond
	defer func() { probeBudget = oldBudget }()
	stubProviders(t, slowProvider{})

	rows := make([]tui.Row, 24)
	for i := range rows {
		rows[i] = tui.Row{Quality: lx.QFlac24, Size: 55158550, Song: &lx.Song{Source: "slow"}}
	}
	start := time.Now()
	measured, below, unmeasured, missedOf := servedRows(context.Background(), rows, lx.QAuto)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("servedRows ran %v; it must stop at probeBudget, not at the caller's deadline", elapsed)
	}
	// Nothing was answered, so nothing may have been claimed as measured, and no
	// row may have been rewritten from a catalogue number it never earned.
	if measured != 0 || below != 0 || unmeasured != len(rows) {
		t.Errorf("(measured, below, unmeasured) = (%d, %d, %d), want (0, 0, %d)",
			measured, below, unmeasured, len(rows))
	}
	if rows[0].Quality != lx.QFlac24 || rows[0].Size != 55158550 {
		t.Errorf("an unanswered row was rewritten: %s/%d", rows[0].Quality, rows[0].Size)
	}
	// The note names the platforms, so the caller can say which rows are unverified.
	if len(missedOf) != 1 || missedOf[0] != "slow24行" {
		t.Errorf("missedOf = %v, want one entry naming the platform and its rows", missedOf)
	}
}

// TestServedRowsReports pins the three counts and that an answer lands on the
// row: the answered rows are the ones measured, a lower tier is a downgrade, and
// the hung rows are neither.
func TestServedRowsReports(t *testing.T) {
	stubProviders(t, answerProvider{pl: &lx.PlayerURL{Quality: lx.Q128}})
	rows := []tui.Row{
		{Song: &lx.Song{Source: "ans", Qualities: []lx.Quality{
			{Type: lx.Q128, Size: 4018053}, {Type: lx.QFlac24, Size: 55158550}}}, Quality: lx.QFlac24, Size: 55158550},
		{Song: &lx.Song{Source: "ans"}},
		{Song: nil},
	}
	measured, below, unmeasured, _ := servedRows(context.Background(), rows, lx.QAuto)
	if measured != 2 || below != 1 || unmeasured != 0 {
		t.Errorf("(measured, below, unmeasured) = (%d, %d, %d), want (2, 1, 0)", measured, below, unmeasured)
	}
	if rows[0].Quality != lx.Q128 || rows[0].Size != 4018053 {
		t.Errorf("the answered row = %s/%d, want 128k and the 128k catalogue size", rows[0].Quality, rows[0].Size)
	}
	// A row that advertised no tier at all learns what the gateway gives, without
	// being counted as a downgrade - it claimed nothing to fall below.
	if rows[1].Quality != lx.Q128 {
		t.Errorf("a row with no catalogue = %q, want the measured tier", rows[1].Quality)
	}
}

// TestUserSourceBreaker pins the circuit breaker. A user source is a script
// hosted somewhere and times out at 25 s per call, so two consecutive misses are
// half a minute paid before the built-in is asked at all - and then again on
// every row of every search. One refusal is not a dead script, so one miss must
// not trip it.
func clearFails(src string) {
	userSourceFailsMu.Lock()
	delete(userSourceFails, src)
	userSourceFailsMu.Unlock()
}

func TestUserSourceBreaker(t *testing.T) {
	const src = "breaker"
	clearFails(src)
	defer clearFails(src)
	if userSourceBlocked(src) {
		t.Fatal("a source that has never been asked must not be blocked")
	}
	noteUserSource(src, false)
	if userSourceBlocked(src) {
		t.Fatal("one refusal must not block the source")
	}
	noteUserSource(src, false)
	if !userSourceBlocked(src) {
		t.Fatal("two consecutive refusals must block the source")
	}
	noteUserSource(src, true)
	if userSourceBlocked(src) {
		t.Fatal("a success must clear the count so a recovered script serves again")
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

// TestFormatMismatch pins the check that turns a mismatched transfer into a
// failure. The load-bearing cases are the migu one first - a flac24bit row that
// lands as a 3.8MB mp3 - and the ogg case, where a 128k promise landing in
// another lossy container is the same rendition and must not fail.
func TestFormatMismatch(t *testing.T) {
	cases := []struct {
		name        string
		advertised  string
		stored      string
		wantFail    bool
		wantContain string
	}{
		{name: "lossless promised, mp3 landed", advertised: lx.QFlac24, stored: ".mp3", wantFail: true,
			wantContain: "列表标注 flac24bit，实际写入的是有损 .mp3 文件"},
		{name: "flac promised, mp3 landed", advertised: lx.QFlac, stored: ".mp3", wantFail: true,
			wantContain: "有损 .mp3"},
		{name: "wav promised, mp3 landed", advertised: lx.QWav, stored: ".mp3", wantFail: true,
			wantContain: "有损 .mp3"},
		{name: "promise met", advertised: lx.QFlac24, stored: ".flac"},
		{name: "wav promise met", advertised: lx.QWav, stored: ".wav"},
		{name: "ape promise met", advertised: lx.QApe, stored: ".ape"},
		{name: "128k promise met", advertised: lx.Q128, stored: ".mp3"},
		{name: "same rendition in another container", advertised: lx.Q128, stored: ".ogg"},
		{name: "320k as aac", advertised: lx.Q320, stored: ".aac"},
		{name: "a surprise upgrade is not a failure", advertised: lx.Q128, stored: ".flac"},
		{name: "auto is not a promise", advertised: lx.QAuto, stored: ".mp3"},
		{name: "best is not a promise", advertised: lx.QBestKnown, stored: ".mp3"},
		{name: "an unlabelled row is not a promise", advertised: "", stored: ".mp3"},
		{name: "a dash row is not a promise", advertised: "-", stored: ".mp3"},
		{name: "an unknown tier is skipped", advertised: "wav24", stored: ".mp3"},
		{name: "no extension to read", advertised: lx.QFlac24, stored: ""},
		{name: "an unrecognized extension is skipped", advertised: lx.QFlac24, stored: ".txt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatMismatch(c.advertised, c.stored)
			if (got != "") != c.wantFail {
				t.Fatalf("formatMismatch(%q, %q) = %q, want failure=%v", c.advertised, c.stored, got, c.wantFail)
			}
			if c.wantFail && !strings.Contains(got, c.wantContain) {
				t.Errorf("message %q does not name the promise and what landed", got)
			}
		})
	}
}

// TestStoredExt pins that the URL decides the format, not the file name. A
// platform can name a file .flac while serving mp3 bytes; trusting the name is
// exactly how the mismatch would slip past the check.
func TestStoredExt(t *testing.T) {
	cases := []struct {
		name string
		url  string
		path string
		want string
	}{
		{name: "the url wins over the file name",
			url: "http://h/t/MP3_128_16_Stero.mp3?t=1", path: "/o/星月神话.flac", want: ".mp3"},
		{name: "a kuwo flac rendition",
			url:  "http://car.kuwo.cn/x/r/F000000B3.flac?bitrate$2000&format$flac&source$x.apk",
			path: "/o/a.mp3", want: ".flac"},
		{name: "no extension in the url falls back to the path",
			url: "http://h/t/vkey=abc", path: "/o/a.MP3", want: ".mp3"},
		{name: "an empty url falls back to the path",
			url: "", path: "/o/a.flac", want: ".flac"},
		{name: "no extension anywhere",
			url: "http://h/t/vkey=abc", path: "/o/a", want: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := storedExt(&lx.PlayerURL{URL: c.url}, c.path); got != c.want {
				t.Errorf("storedExt = %q, want %q", got, c.want)
			}
		})
	}
}

// TestAudioExtPathOnly is the regression behind PathExt: only the path may be
// read, because a query string can carry its own extension.
func TestAudioExtPathOnly(t *testing.T) {
	for _, c := range []struct {
		url  string
		want string
	}{
		{"/t/a.mp3?$format$flac&x.apk", ".mp3"},
		{"/t/a.mp3?x.flac", ".mp3"},
		{"/t/a.mp3#f.flac", ".mp3"},
		{"/t/.hidden.mp3", ".mp3"},
		{"/t/noext", ""},
		{"", ""},
		{"/t/a.apk", ""},
	} {
		if got := lx.PathExt(c.url); got != c.want {
			t.Errorf("PathExt(%q) = %q, want %q", c.url, got, c.want)
		}
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
