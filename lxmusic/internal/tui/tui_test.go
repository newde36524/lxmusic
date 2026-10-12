package tui

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestRuneWidth pins the East Asian Width behaviour that keeps the table
// columns aligned when Chinese text is involved.
func TestRuneWidth(t *testing.T) {
	cases := map[rune]int{
		'A': 1, '0': 1, '.': 1, '-': 1, ' ': 1, '…': 1, // U+2026 is one cell
		'京': 2, '周': 2, '杰': 2, '伦': 2, '晴': 2, '天': 2,
		'、': 2, '。': 2, '（': 2, '）': 2, '「': 2, '」': 2,
		'！': 2, '？': 2, 'Ａ': 2, '＃': 2,
		'あ': 2, 'ア': 2, 'ｶ': 1, '가': 2,
		'㐀': 2, '豈': 2, '豕': 2,
		'\x00': 0, '\x07': 0, '\x1b': 0, 0x7f: 0,
	}
	for r, want := range cases {
		if got := rw(r); got != want {
			t.Errorf("rw(%q U+%04X) = %d, want %d", r, r, got, want)
		}
	}
}

func TestDisp(t *testing.T) {
	for _, c := range []struct {
		s    string
		want int
	}{
		{"晴天", 4},
		{"Jay Chou", 8},
		{"周杰伦 - 晴天", 13}, // 5 CJK = 10, plus space hyphen space = 3
		{"周杰伦", 6},
		{"", 0},
		{"\x1b[32m", 0}, // an escape sequence takes no cells
		{"\x1b[32m晴天\x1b[0m", 4},
		{"\x1b[2K", 0}, // truncated sequence still counts as one
	} {
		if got := disp(c.s); got != c.want {
			t.Errorf("disp(%q) = %d, want %d", c.s, got, c.want)
		}
	}
}

func TestClipNoSplit(t *testing.T) {
	// Clipping must never let the visible width exceed the cap, and never cut a
	// two-cell rune in half.
	for _, s := range []string{"晴天", "周杰伦的晴天", "晴天abc", "abc晴天", "1234567890"} {
		for w := 0; w <= disp(s)+2; w++ {
			got := clip(s, w)
			if d := disp(got); d > w {
				t.Errorf("clip(%q,%d) = %q has width %d > %d", s, w, got, d, w)
			}
			if d := disp(s); d <= w && got != s {
				t.Errorf("clip(%q,%d) = %q, input already fit", s, w, got)
			}
		}
	}
}

// TestTableFrame pins the contract the picker depends on: exactly one line per
// row, nothing wider than the terminal, and the header's column boundaries
// identical to every body row's. That last one is the regression that had the
// captions sitting to the right of the values they labelled: rendering
// StyleNone let the header row run wider than the body rows.
func TestTableFrame(t *testing.T) {
	rows := []Row{
		{Index: 1, Platform: "kw", Name: "晴天", Artist: "周杰伦", Duration: "04:29", Quality: "320k", Size: 10290000},
		{Index: 2, Platform: "kg", Name: strings.Repeat("很长的歌名", 40), Artist: "周杰伦、林俊杰、其他很多歌手", Duration: "120:00", Quality: "flac24bit", Size: 640000000},
		{Index: 3, Platform: "wy", Name: "晴天(原版)", Artist: "周杰伦", Duration: "04:29", Quality: "flac", Size: 0},
	}
	for cols := pickerMin; cols <= 180; cols += 13 {
		w := computeWidths(rows, cols, asciiChrome)
		rule, head, body := tableFrame(rows, w)
		if head == "" || rule == "" {
			t.Fatalf("cols=%d: tableFrame returned no header or rule", cols)
		}
		if len(body) != len(rows) {
			t.Fatalf("cols=%d: tableFrame returned %d rows for %d", cols, len(body), len(rows))
		}
		want := pipeCells(head)
		if len(want) != len(headers)+1 {
			t.Fatalf("cols=%d: header has %d pipes, want %d", cols, len(want), len(headers)+1)
		}
		for _, ln := range append([]string{rule}, append([]string{head}, body...)...) {
			if d := disp(ln); d > cols {
				t.Fatalf("cols=%d: a line is %d cells wide: %q", cols, d, ln)
			}
		}
		for _, ln := range body {
			if got := pipeCells(ln); !sameInts(got, want) {
				t.Fatalf("cols=%d: column boundaries drifted from the header: %v vs %v\n%s", cols, got, want, ln)
			}
		}
	}
}

// pipeCells gives the cell offset of each vertical rule, so two lines can be
// compared column by column regardless of how many cells the content uses.
func pipeCells(s string) []int {
	var out []int
	pos := 0
	for _, r := range s {
		if r == '|' {
			out = append(out, pos)
		}
		pos += rw(r)
	}
	return out
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestPrintTableWidth checks the bordered variant too, which is the one piped to
// a file or a less wide terminal.
func TestPrintTableWidth(t *testing.T) {
	rows := []Row{
		{Index: 1, Platform: "网易云音乐", Name: "晴天", Artist: "周杰伦", Duration: "04:29", Quality: "320k", Size: 4330000},
		{Index: 2, Platform: "tx", Name: strings.Repeat("很长的歌名", 40), Artist: "周杰伦", Duration: "120:00", Quality: "flac24bit", Size: 999454720},
	}
	var buf strings.Builder
	PrintTable(&buf, 0, "搜索 \"晴天\" — 2 条结果", rows)
	for i, ln := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if d := disp(ln); d > 80 {
			t.Fatalf("line %d is %d cells wide: %q", i+1, d, ln)
		}
	}
}

func TestLayoutBounds(t *testing.T) {
	rows := []Row{
		{Index: 1, Platform: "kw", Name: "晴天", Artist: "周杰伦", Duration: "04:29", Quality: "320k", Size: 10290000},
		{Index: 2, Platform: "kg", Name: "晴天(原版)", Artist: "周杰伦", Duration: "04:29", Quality: "128k", Size: 4200000},
		{Index: 3, Platform: "hyw", Name: "晴天", Artist: "周杰伦", Duration: "04:29", Quality: "flac", Size: 32430000},
	}
	// A two-line status is what makes chromeFor matter: the visible row count has
	// to shrink rather than the screen growing past the bottom. It happens two
	// ways - the confirm prompt, and a long error message that wraps.
	statuses := []string{
		"",
		"正在下载 kw:1 …",
		"✗ kw:1  失败",
		"下载 晴天 - 周杰伦?\n保存到 C:\\Users\\jmrx\\Downloads\\晴天.flac    [y] 下载  [n] 取消",
		"解析保存路径 kw:1 …",
		"✗ kw:1  kugou gives no anonymous audio url, so add a user source that " +
			"serves kg or use a paid account, which is what the settings page does",
		"✗ " + strings.Repeat("x", 400),
	}
	for cols := pickerMin; cols <= 200; cols += 17 {
		for r := pickRowsMin; r <= 40; r += 7 {
			for cur := 0; cur <= len(rows); cur++ {
				for top := 0; top <= len(rows); top++ {
					for _, st := range statuses {
						lines := layout(Prompt{Title: "T", Rows: rows, Foot: "F"}, cols, r, cur, top, st)
						if len(lines) != r {
							t.Fatalf("cols=%d rows=%d: layout returned %d lines for a %d-row terminal",
								cols, r, len(lines), r)
						}
						for i, l := range lines {
							if d := disp(l); d > cols {
								t.Fatalf("cols=%d rows=%d line %d is %d wide: %q",
									cols, r, i, d, l)
							}
						}
					}
				}
			}
		}
	}
}

// TestSizeCell pins the size column formatting, including the unknown case,
// which must not read as a tiny real file.
func TestSizeCell(t *testing.T) {
	for _, c := range []struct {
		bytes int64
		want  string
	}{
		{0, "-"},
		{-1, "-"},
		{1024, "1.00KB"},
		{1024 * 1024, "1.00MB"},
		{10290000, "9.81MB"},
		{32430000, "30.93MB"},
		{999454720, "953.15MB"},
	} {
		if got := sizeCell(c.bytes); got != c.want {
			t.Errorf("sizeCell(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

// TestChromeAgrees pins that chromeFor and statusBlock describe the same status
// block. layout uses one for its row budget and the other for its content, so a
// disagreement silently either overflows the screen or wastes a row. It sweeps
// widths too, because wrapping is what makes the two disagree: a status that
// wraps to two lines must reserve two rows, and one that fits a single line must
// not be charged for the second.
func TestChromeAgrees(t *testing.T) {
	statuses := []string{"", "a", "a\nb", "a\nb\nc", "first\n"}
	for cols := 0; cols <= 140; cols += 7 {
		for _, st := range statuses {
			if got, want := chromeFor(st, cols)-fixedChrome, len(statusBlock(st, cols)); got != want {
				t.Errorf("cols=%d chromeFor(%q) accounts for %d status line(s), statusBlock emitted %d",
					cols, st, got, want)
			}
		}
	}
}

// TestWrapStatus is the regression for the truncated download error: the reason a
// download failed used to be clipped to one terminal width, so the remedy was
// invisible. Whatever fits in maxStatusLines lines must appear, and only the
// overflow is cut.
func TestWrapStatus(t *testing.T) {
	for _, c := range []struct {
		status string
		cols   int
		want   int // number of lines
	}{
		{"", 80, 1},
		{"short", 80, 1},
		{"a\nb", 80, 2},
		{strings.Repeat("w ", 100), 80, 3},  // more than a screen of text, no newline
		{strings.Repeat("周杰伦 ", 60), 80, 3}, // CJK wraps without spaces
		{strings.Repeat("x", 500), 40, 3},   // no break point at all
		{"a b c d\n" + strings.Repeat("e ", 60), 20, 3},
	} {
		got := wrapStatus(c.status, c.cols)
		if len(got) != c.want {
			t.Errorf("wrapStatus(%q,%d) = %d lines, want %d: %q", c.status, c.cols, len(got), c.want, got)
		}
		if len(got) > maxStatusLines {
			t.Errorf("wrapStatus(%q,%d) emitted %d lines, more than maxStatusLines %d",
				c.status, c.cols, len(got), maxStatusLines)
		}
		for i, ln := range got {
			if d := disp(ln); d > c.cols {
				t.Errorf("wrapStatus(%q,%d) line %d is %d wide: %q", c.status, c.cols, i, d, ln)
			}
		}
	}
}

// TestWrapStatusMarksOverflow makes sure the cut is visible: a message that needs
// more than maxStatusLines lines must say so on its last line.
func TestWrapStatusMarksOverflow(t *testing.T) {
	long := "✗ " + strings.Repeat("x", 400)
	for _, cols := range []int{20, 40, 80, 120} {
		got := wrapStatus(long, cols)
		last := got[len(got)-1]
		if !strings.HasSuffix(last, "..") {
			t.Errorf("cols=%d: overflow was cut without the .. marker: %q", cols, last)
		}
	}
	// Everything that fits must not be marked.
	if got := wrapStatus("short message", 80); !strings.HasSuffix(got[0], "message") {
		t.Errorf("fitting text was marked as cut: %q", got)
	}
}

// TestCutAtWordBoundary checks the break position: English breaks at a space,
// CJK has none and breaks at the cell boundary, and the split never splits a
// rune or runs past the width.
func TestCutAtWordBoundary(t *testing.T) {
	cases := []struct {
		s     string
		width int
		fit   string
	}{
		{"a b c d", 5, "a b"},
		{"hello world", 11, "hello world"},
		{"hello world", 5, "hello"},
		{"周杰伦的晴天", 5, "周杰"},
		{"晴天abc", 4, "晴天"}, // two wide runes are exactly the budget
		{"abc", 3, "abc"},
	}
	for _, c := range cases {
		fit, rest := cutAt(c.s, c.width)
		if fit != c.fit {
			t.Errorf("cutAt(%q,%d) fit = %q, want %q", c.s, c.width, fit, c.fit)
		}
		if disp(fit) > c.width {
			t.Errorf("cutAt(%q,%d) fit is %d wide, want at most %d", c.s, c.width, disp(fit), c.width)
		}
		// The break space is consumed by the split, so the remainder is the rest
		// of the input with its leading space removed rather than appended.
		if want := strings.TrimLeft(c.s[len(fit):], " "); rest != want {
			t.Errorf("cutAt(%q,%d) rest = %q, want %q", c.s, c.width, rest, want)
		}
	}
	if fit, rest := cutAt("abc", 0); fit != "" || rest != "abc" {
		t.Errorf("cutAt width 0 = %q/%q, want nothing cut", fit, rest)
	}
}

// TestFailureStatus checks that a failed download names the row once. The
// providers interpolate s.KeyString() into their errors, so prefixing the ref
// used to print "kg:<hash>  kg: kg:<hash> ..." - the ref twice, and the reason
// pushed off the end of the line.
func TestFailureStatus(t *testing.T) {
	if got := failureStatus("kw:1", errors.New("connection reset")); !strings.Contains(got, "kw:1") {
		t.Errorf("a bare error lost the reference: %q", got)
	}
	err := errors.New("kg:abc is a paid track (privilege=10 pay_type=3 price=200)")
	got := failureStatus("kg:abc", err)
	if strings.Count(got, "kg:abc") != 1 {
		t.Errorf("the reference was repeated: %q", got)
	}
	if !strings.Contains(got, "✗ ") {
		t.Errorf("missing the error marker: %q", got)
	}
	if statusBlock(got, 80)[0] == "" {
		t.Errorf("a failure status must not render empty: %q", got)
	}
}

// TestFailureStatusReadable is the end-to-end regression for the screenshot that
// started this: a user picked a kugou row, confirmed the download, and got an
// error that clipped at "kugou gives no an.." with the remedy unreadable. Every
// character of the message must reach the screen across the two status lines.
func TestFailureStatusReadable(t *testing.T) {
	msg := "kg:154265869D19C7899C89B58E8B55782F5CDF9 is a paid track " +
		"(privilege=10 pay_type=3 price=200): kugou gives no anonymous " +
		"audio url, so add a user source that serves kg or use a paid account"
	ref := "kg:154265869D19C7899C89B58E8B55782F5CDF9"
	st := failureStatus(ref, errors.New(msg))
	if n := strings.Count(st, ref); n != 1 {
		t.Fatalf("the reference appears %d times, want 1: %q", n, st)
	}
	// A wide terminal holds the whole message across the two status lines. This
	// is the case from the screenshot, where the remedy was clipped away.
	wide := statusBlock(st, 160)
	if len(wide) != 2 {
		t.Errorf("cols=160: rendered %d lines, want 2: %q", len(wide), wide)
	}
	for i, ln := range wide {
		if d := disp(ln); d > 160 {
			t.Errorf("cols=160: line %d is %d cells wide: %q", i, d, ln)
		}
		if strings.Contains(ln, "..") {
			t.Errorf("cols=160: line %d was cut though the message fits in two: %q", i, ln)
		}
	}
	// A narrow terminal has less room than the message, so the cut must be marked
	// rather than dropped silently. 40 cells hold about two of the three status
	// lines, so this one still overflows.
	narrow := statusBlock(st, 40)
	if len(narrow) > maxStatusLines {
		t.Errorf("cols=40: rendered %d lines, more than maxStatusLines", len(narrow))
	}
	if !strings.Contains(narrow[len(narrow)-1], "..") {
		t.Errorf("cols=40: cut %d cells without the .. marker: %q", disp(st), narrow)
	}
	// The row budget must always match the rendered block.
	for _, cols := range []int{68, 80, 120, 160, 200} {
		if got := chromeFor(st, cols) - fixedChrome; got != len(statusBlock(st, cols)) {
			t.Errorf("cols=%d: chromeFor reserved %d rows, statusBlock rendered %d",
				cols, got, len(statusBlock(st, cols)))
		}
	}
}

// TestWidthsHideNothing is the guarantee behind the dynamic width: given as much
// room as the content needs, no cell is truncated. The old fixed floors clipped
// long song titles and long artist lists at 80 columns.
func TestWidthsHideNothing(t *testing.T) {
	rows := []Row{
		{Index: 1, Platform: "网易云音乐", Name: "晴天", Artist: "周杰伦", Duration: "04:29", Quality: "flac", Size: 52830000},
		{Index: 2, Platform: "酷狗音乐", Name: strings.Repeat("很长的歌名", 6),
			Artist: "周杰伦、林俊杰、其他很多歌手、还有一些", Duration: "120:00", Quality: "flac24bit", Size: 999454720},
		{Index: 3, Platform: "tx", Name: "晴天(原版)", Artist: "周杰伦", Duration: "04:29", Quality: "-", Size: 0},
	}
	ideal := columnWidths(rows)
	for _, cols := range []int{ideal.total(), ideal.total() + 17, 200} {
		w := computeWidths(rows, cols, noneChrome)
		// It must fill the terminal rather than waste it.
		if w.total() != cols {
			t.Errorf("cols=%d: widths total %d, want to fill %d exactly", cols, w.total(), cols)
		}
		ws := w.all()
		for i, h := range headers {
			if ws[i] < disp(h)+cellPad {
				t.Errorf("cols=%d: column %d (%s) is %d wide, narrower than its own header",
					cols, i, h, ws[i])
			}
			if disp(h) > ws[i]-cellPad {
				t.Errorf("cols=%d: header %q does not fit its own column", cols, h)
			}
		}
		for _, r := range rows {
			for i, v := range cellText(r) {
				if disp(v) > ws[i]-cellPad {
					t.Errorf("cols=%d: %q does not fit column %d (%s), content would be hidden",
						cols, v, i, headers[i])
				}
			}
		}
	}
}

// TestWidthsFitNarrow checks the other end: squeezed into the minimum terminal
// the table still fits on screen, and no column loses its own caption. When the
// content is intrinsically wider than the terminal some column must still be
// trimmed, but only after every column has been reduced to its floor - so the
// invariants are that the total fits and the floors hold, not that everything is
// at its floor. The floor starts at minCols + chrome: a bordered table needs the
// column floors plus its own border to fit at all.
func TestWidthsFitNarrow(t *testing.T) {
	rows := []Row{
		{Index: 20, Platform: "网易云音乐", Name: strings.Repeat("很长的歌名", 20),
			Artist: "周杰伦、林俊杰", Duration: "120:00", Quality: "flac24bit", Size: 999454720},
	}
	floors := headerFloors()
	for _, chrome := range []int{noneChrome, asciiChrome} {
		for cols := minCols + chrome; cols <= minCols+chrome+60; cols += 3 {
			w := computeWidths(rows, cols, chrome)
			if w.total()+chrome > cols {
				t.Fatalf("cols=%d chrome=%d: widths total %d, table would run %d cells past the terminal",
					cols, chrome, w.total(), w.total()+chrome-cols)
			}
			ws := w.all()
			for i, f := range floors {
				if ws[i] < f {
					t.Fatalf("cols=%d: column %d is %d wide, below its header floor %d", cols, i, ws[i], f)
				}
			}
		}
	}
}

// TestColumnWidthsMeasure tracks the widest value in each column, not just the
// header, which is what makes a 60 MB size line up under a 6-cell caption.
func TestColumnWidthsMeasure(t *testing.T) {
	rows := []Row{{Index: 7, Platform: "kw", Name: "晴天", Artist: "周杰伦",
		Duration: "04:29", Quality: "flac", Size: 52830000}}
	w := columnWidths(rows)
	if w.size < disp(sizeCell(52830000))+cellPad {
		t.Errorf("size column is %d, want at least %d to hold %q",
			w.size, disp(sizeCell(52830000))+cellPad, sizeCell(52830000))
	}
	if w.platform < disp("平台名称")+cellPad {
		t.Errorf("platform column is %d, narrower than its header", w.platform)
	}
}

// TestShrinkProtectsMeasuredValues pins the trim order. When the terminal cannot
// hold everything, the free-form title and artist give up width while the
// measured fields keep theirs - a cut quality label or file size is a wrong
// answer, which is worse than a cut title.
func TestShrinkProtectsMeasuredValues(t *testing.T) {
	rows := []Row{
		{Index: 20, Platform: "网易云音乐", Name: strings.Repeat("很长的歌名", 6),
			Artist: "周杰伦、林俊杰、其他很多歌手", Duration: "120:00", Quality: "flac24bit", Size: 999454720},
	}
	for cols := minCols + asciiChrome; cols <= 120; cols += 7 {
		w := computeWidths(rows, cols, asciiChrome)
		ws := w.all()
		ideal := columnWidths(rows).all()
		// The measured fields are only allowed to shrink if the free text is
		// already down to its caption width.
		floor := headerFloors()
		textRoom := 0
		for _, i := range []int{2, 4} {
			if ideal[i]-floor[i] > 0 {
				textRoom += ideal[i] - floor[i]
			}
		}
		if sumOf(ideal)+asciiChrome-cols <= textRoom {
			// The free text alone can absorb the overage, so the measured
			// columns must still be showing their full values.
			for _, i := range []int{0, 3, 5, 6} {
				if ws[i] < ideal[i] {
					t.Errorf("cols=%d: measured column %d shrank from %d to %d before the free text did",
						cols, i, ideal[i], ws[i])
				}
			}
		}
		// And nothing ever loses its caption.
		for i, f := range floor {
			if ws[i] < f {
				t.Errorf("cols=%d: column %d is %d wide, below its caption width %d", cols, i, ws[i], f)
			}
		}
	}
}

func TestBar(t *testing.T) {
	for _, c := range []struct {
		frac   float64
		filled int
		pct    int
	}{
		{-5, 0, 0}, {0, 0, 0}, {0.25, 5, 25}, {0.5, 10, 50}, {0.75, 15, 75}, {1, 20, 100}, {9, 20, 100},
	} {
		got := bar(c.frac, 20)
		if f := strings.Count(got, "█"); f != c.filled {
			t.Errorf("bar(%v) has %d filled cells, want %d: %q", c.frac, f, c.filled, got)
		}
		if u := strings.Count(got, "░"); u != 20-c.filled {
			t.Errorf("bar(%v) has %d empty cells, want %d: %q", c.frac, u, 20-c.filled, got)
		}
		if !strings.Contains(got, fmt.Sprintf("%3d%%", c.pct)) {
			t.Errorf("bar(%v) = %q, missing %d%%", c.frac, got, c.pct)
		}
		if d := disp(got); d != 27 {
			t.Errorf("bar(%v) is %d cells wide, want the fixed 27: %q", c.frac, d, got)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	for _, c := range []struct {
		n    int64
		want string
	}{
		{0, "0.00KB"}, {1024, "1.00KB"}, {10290000, "9.81MB"}, {999454720, "953.15MB"},
	} {
		if got := humanBytes(c.n); got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

// TestProgressLine covers the two shapes the bar can take: with a total from
// Content-Length, and without, where a percentage would be a lie.
func TestProgressLine(t *testing.T) {
	if got := (&DownloadProgress{}).Line("kw:1"); got != "正在解析 kw:1 …" {
		t.Errorf("before any report = %q, want the resolve phase", got)
	}
	if got := (*DownloadProgress)(nil).Line("kw:1"); got != "正在解析 kw:1 …" {
		t.Errorf("nil progress = %q", got)
	}
	p := &DownloadProgress{}
	p.Report(1000000, 2000000)
	if got := p.Line("kw:1"); !strings.Contains(got, "50%") || !strings.Contains(got, "/") {
		t.Errorf("halfway with a total = %q, want a bar and a fraction", got)
	}
	p.Report(2000000, 2000000)
	if got := p.Line("kw:1"); !strings.Contains(got, "100%") {
		t.Errorf("complete = %q, want 100%%", got)
	}
	// No Content-Length: a byte count, never a percentage.
	q := &DownloadProgress{}
	q.Report(3000000, 0)
	got := q.Line("tx:9")
	if strings.Contains(got, "%") {
		t.Errorf("without a total = %q, must not show a percentage", got)
	}
	if !strings.Contains(got, "2.86MB") {
		t.Errorf("without a total = %q, want the byte count", got)
	}
	// Reporting backwards or with a smaller total must not regress the bar.
	p.Report(0, 0)
	if got := p.Line("kw:1"); !strings.Contains(got, "100%") {
		t.Errorf("after a stale report = %q, want to still show 100%%", got)
	}
}

// TestProgressResetAcrossDownloads is the picker's real shape: one counter for
// the whole session, handed to every row. Report only ever raises done and
// total, so a finished download's 100% is unresettable on its own - the second
// song would open on the first one's full bar and never move.
func TestProgressResetAcrossDownloads(t *testing.T) {
	p := &DownloadProgress{}

	p.Report(10, 10)
	if got := p.Line("kw:1"); !strings.Contains(got, "100%") {
		t.Fatalf("first download complete = %q, want 100%%", got)
	}

	p.Reset()
	if got := p.Line("kw:2"); got != "正在解析 kw:2 …" {
		t.Errorf("fresh counter = %q, want the resolve phase", got)
	}

	// A smaller file, and the bar must fill on its own terms.
	p.Report(5000000, 10000000)
	if got := p.Line("kw:2"); !strings.Contains(got, "50%") {
		t.Errorf("halfway into the second download = %q, want 50%%", got)
	}
	p.Report(10000000, 10000000)
	if got := p.Line("kw:2"); !strings.Contains(got, "100%") {
		t.Errorf("second download complete = %q, want 100%%", got)
	}

	// A late report from the first download must not push the second one back.
	p.Report(10, 10)
	if got := p.Line("kw:2"); !strings.Contains(got, "100%") {
		t.Errorf("after a stale report = %q, want 100%%", got)
	}

	// And an unknown-length stream must not inherit the finished total either.
	p.Reset()
	p.Report(3000000, 0)
	if got := p.Line("tx:9"); strings.Contains(got, "%") {
		t.Errorf("without a total = %q, must not show a percentage", got)
	}

	// Reset on a nil receiver is as harmless as Line on one.
	(*DownloadProgress)(nil).Reset()
}

// TestProgressResetConcurrent pins that Reset takes the same lock as Report and
// Line, so clearing between two reads cannot publish a torn state.
func TestProgressResetConcurrent(t *testing.T) {
	p := &DownloadProgress{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < 200; n++ {
				if i%2 == 0 {
					p.Reset()
				} else {
					p.Report(int64(n), int64(n)*2)
					p.Line("kw:1")
				}
			}
		}(i)
	}
	wg.Wait()
}
