// Package tui draws an interactive search-result table for a plain terminal:
// the arrow keys move the selection, Enter asks whether to download, and a
// progress bar tracks the transfer. q/Esc quits.
//
// Column widths, padding and alignment come from olekukonko/tablewriter, which
// is why the width maths in this package is expressed in declared column widths
// rather than in cell text. The screen itself is still painted with handwritten
// ANSI escapes, and the keyboard comes from a thin platform layer that uses
// termios on unix and ReadConsoleInputW on windows.
package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	lx "lxmusic/internal/lx"
	"unicode"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter/tw"
)

const (
	// fixedChrome is how many rows the non-data furniture that never varies
	// occupies: title, spacer, rule, header, rule, closing rule, spacer and
	// footer. The status block then adds one more row, or two while a download is
	// being confirmed, so the visible row count is derived rather than assumed.
	// This must equal the non-data lines layout emits; TestChromeAgrees and
	// TestLayoutBounds both catch a drift in either direction.
	fixedChrome = 8

	// maxStatusLines is the status block's tallest. The confirm prompt uses two
	// - the question, then the destination - and a long error message wraps to
	// two as well, so an explanation fits without being cut off.
	maxStatusLines = 2

	// pickRowsMin is the narrowest terminal the picker is drawn at in rows: the
	// furniture, one data row, and the status block at its tallest. Below it there
	// is no layout that fits, and drawing anyway is what pushed the footer off the
	// bottom of the screen. Derived so a change to fixedChrome moves it with it.
	pickRowsMin = fixedChrome + maxStatusLines + 1

	// progressPoll is how often a running download is redrawn. Fast enough to
	// look live, slow enough that the redraw cost never matters.
	progressPoll = 100 * time.Millisecond

	cReset = "\x1b[0m"
	cBold  = "\x1b[1m"
	cDim   = "\x1b[2m"
	cRed   = "\x1b[31m"
	cCyan  = "\x1b[36m"
	cRev   = "\x1b[7m"
)

// Row is one line of the picker table. The display fields are rendered as-is;
// Song is the record they came from, and is the one that gets downloaded, so a
// download never has to re-look-up a song the table already has in hand.
type Row struct {
	Index    int
	Platform string
	Name     string
	Artist   string
	Duration string
	Quality  string
	Size     int64  // bytes the platform announced for Quality, 0 = unknown
	Ref      string // canonical "source:id" reference

	Song *lx.Song
}

// Key is a normalised key event.
type Key int

const (
	KeyNone Key = iota
	KeyUp
	KeyDown
	KeyPageUp
	KeyPageDown
	KeyEnter
	KeyYes
	KeyNo
	KeyQuit
)

// DownloadProgress carries the byte counts a download reports as they arrive.
// The download may report on every read; the picker polls on a timer, so the
// two never have to agree on a cadence.
type DownloadProgress struct {
	mu    sync.Mutex
	done  int64
	total int64
	start time.Time
	verb  string // 正在下载, once the first byte has arrived
}

// Report is called by the download with the bytes written so far and the total
// the server announced, if it announced one. The first call also marks the
// transfer as started: until then the line reports that the url is still being
// resolved, which is where most of a user source's time goes.
func (p *DownloadProgress) Report(done, total int64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.start.IsZero() {
		p.start = time.Now()
	}
	if done > p.done {
		p.done = done
		p.verb = "正在下载"
	}
	if total > p.total {
		p.total = total
	}
}

// Reset clears a finished download, so the next song in the same session does
// not inherit its byte count, total, and start time. The picker keeps one
// DownloadProgress for the whole session and hands it to every row, and Report
// only ever raises done and total, so a non-zero baseline can never come back
// down: the second download would open on the first one's full bar and never
// move, and its rate would be divided by an elapsed time that started with the
// previous song.
func (p *DownloadProgress) Reset() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done = 0
	p.total = 0
	p.start = time.Time{}
	p.verb = ""
}

// Line renders the transfer as one screen line: a bar when the server gave a
// total, a plain byte count otherwise.
func (p *DownloadProgress) Line(label string) string {
	// A nil counter renders like an empty one: no bytes in, so still resolving.
	// One code path for both keeps the two from drifting apart.
	var done, total int64
	var start time.Time
	var verb string
	if p != nil {
		p.mu.Lock()
		done, total, start, verb = p.done, p.total, p.start, p.verb
		p.mu.Unlock()
	}
	// No byte has arrived, so nothing is downloading yet: the url is still being
	// resolved, and for a user source that is most of the wait. Saying 下载 here
	// would report a transfer that has not happened.
	if verb == "" {
		verb = "正在解析"
	}
	s := verb + " " + label
	if done <= 0 {
		return s + " …"
	}
	if total > 0 {
		s += "  " + bar(float64(done)/float64(total), 20) + "  " +
			humanBytes(done) + " / " + humanBytes(total)
	} else {
		s += "  " + humanBytes(done)
	}
	if !start.IsZero() {
		if el := time.Since(start).Seconds(); el > 0 {
			s += "  " + humanBytes(int64(float64(done)/el)) + "/s"
		}
	}
	return s
}

// bar draws a fixed-width progress bar with the percentage beside it.
func bar(frac float64, n int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(n) + 0.5)
	if filled > n {
		filled = n
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", n-filled) +
		"] " + fmt.Sprintf("%3d%%", int(frac*100+0.5))
}

// humanBytes renders a byte count as KB or MB.
func humanBytes(n int64) string {
	if n < 1024*1024 {
		return fmt.Sprintf("%.2fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.2fMB", float64(n)/(1024*1024))
}

// Prompt describes one picker session.
type Prompt struct {
	Title string
	Rows  []Row
	Foot  string
	// Download is called for the selected row after it is confirmed. The picker
	// stays up afterwards, so several songs can be grabbed in one session. The
	// context is cancelled if the user aborts with q / Esc / Ctrl+C while a
	// download runs.
	Download func(Row, context.Context) (status string, err error)
	// SavePath reports where the selected row will be written, shown in the
	// confirm prompt so the destination is visible before the transfer starts.
	// It must be cheap: it runs between the Enter keypress and the prompt
	// appearing, so anything with a network round trip in it stalls the UI.
	SavePath func(Row) string
	// Progress, if set, is filled in by the download and polled while it runs.
	Progress *DownloadProgress
}

// Terminal reports whether both standard input and output are attached to a
// real terminal. It is false when either stream is piped, redirected, or when
// running unattended (CI, make, a scripted harness).
func Terminal() bool { return isTerminal() }

// Pick runs the interactive picker and returns the last status line it showed.
// A non-nil error means the terminal could not be taken over, so the caller
// should fall back to a plain table instead of having printed escape garbage.
func Pick(ctx context.Context, p Prompt) (string, error) {
	if len(p.Rows) == 0 {
		return "", nil
	}
	cols, rows := termSize()
	if cols < pickerMin {
		cols = pickerMin
	}
	// One data row plus the confirm prompt's two-line status: below that there is
	// no layout that fits, and drawing anyway is what put the footer off the
	// bottom of the screen.
	if rows < pickRowsMin {
		rows = pickRowsMin
	}
	if err := enable(); err != nil {
		return "", err
	}
	defer disable()

	keys := make(chan Key, 8)
	quit := make(chan struct{})
	go func() {
		for {
			k, err := nextKey()
			if err != nil {
				return
			}
			select {
			case keys <- k:
			case <-quit:
				return
			}
		}
	}()
	defer close(quit)

	cur, top := 0, 0
	status := ""
loop:
	for {
		if err := ctx.Err(); err != nil {
			break
		}
		vis := visRows(rows-chromeFor(status, cols), len(p.Rows))
		render(p, cols, rows, cur, top, status)
		select {
		case <-ctx.Done():
			break loop
		case k := <-keys:
			switch k {
			case KeyUp, KeyDown, KeyPageUp, KeyPageDown:
				cur = moveSelection(k, cur, vis, len(p.Rows))
			case KeyEnter:
				if p.Download != nil {
					enterPhase(p, cols, rows, &cur, &top, &status, keys, ctx, p.Rows[cur])
				}
			case KeyQuit:
				break loop
			}
		}
		fixScroll(&cur, &top, vis, len(p.Rows))
	}
	return status, nil
}

// enterPhase confirms the selected row, then downloads it if the user agrees.
// The confirmation comes first because the download is the slow part, and it is
// easy to have moved one row too far before pressing Enter.
func enterPhase(p Prompt, cols, rows int, cur, top *int, status *string, keys <-chan Key, ctx context.Context, row Row) {
	if !confirmDownload(p, cols, rows, cur, top, status, keys, ctx, row) {
		return
	}
	// The download runs without dropping the table. Keys keep being read while it
	// works, so q / Esc / Ctrl+C can abort it through the context. The progress
	// counter is cleared first: it is shared by every row in the session, and the
	// last download left it at its own 100%.
	p.Progress.Reset()
	dctx, cancel := context.WithCancel(ctx)
	type done struct {
		status string
		err    error
	}
	ch := make(chan done, 1)
	go func() {
		s, e := p.Download(row, dctx)
		cancel()
		ch <- done{s, e}
	}()
	*status = p.Progress.Line(row.Ref)
	tick := time.NewTicker(progressPoll)
	defer tick.Stop()
wait:
	for {
		render(p, cols, rows, *cur, *top, *status)
		select {
		case d := <-ch:
			if d.err != nil {
				*status = failureStatus(row.Ref, d.err)
			} else {
				*status = d.status
			}
			break wait
		case <-tick.C:
			*status = p.Progress.Line(row.Ref)
		case k := <-keys:
			if k == KeyQuit {
				cancel()
				*status = "已取消 " + row.Ref
				break wait
			}
		}
	}
}

// failureStatus is the status text a failed download leaves behind. It repeats
// the row's reference only when the error does not already carry it: several
// providers interpolate s.KeyString() into their message, so prefixing the ref
// unconditionally printed it twice and ate a third of the one status line the
// picker has. That is what hid the reason the user actually needed.
func failureStatus(ref string, err error) string {
	msg := err.Error()
	if strings.Contains(msg, ref) {
		return "✗ " + msg
	}
	return "✗ " + ref + "  " + msg
}

// confirmDownload asks y/n about one row, showing where it will be written. It
// returns false when the user declines; Up/Down decline too and move the
// selection, so backtracking costs one keystroke.
func confirmDownload(p Prompt, cols, rows int, cur, top *int, status *string, keys <-chan Key, ctx context.Context, row Row) bool {
	dst := "(未知)"
	if p.SavePath != nil {
		if s := p.SavePath(row); s != "" {
			dst = s
		}
	}
	*status = fmt.Sprintf("下载 %s - %s?    [y] 下载  [n] 取消", row.Name, row.Artist) +
		"\n保存到 " + dst
	for {
		render(p, cols, rows, *cur, *top, *status)
		select {
		case <-ctx.Done():
			*status = ""
			return false
		case k := <-keys:
			switch k {
			case KeyEnter, KeyYes:
				return true
			case KeyNo, KeyQuit:
				*status = "已取消 " + row.Ref
				return false
			case KeyUp, KeyPageUp:
				if *cur > 0 {
					*cur--
				}
				*status = ""
				return false
			case KeyDown, KeyPageDown:
				if *cur < len(p.Rows)-1 {
					*cur++
				}
				*status = ""
				return false
			}
		}
	}
}

// moveSelection applies one navigation key to the selection and returns the new
// row. The arrows step one row; PgUp/PgDn move a whole page, which is what they
// exist for once the list is long enough to scroll. vis is the number of rows
// on screen and n the number there are, and the selection never leaves [0, n-1]
// - not at either end, not when vis is larger than the list.
func moveSelection(k Key, cur, vis, n int) int {
	if n <= 0 {
		return 0
	}
	switch k {
	case KeyUp:
		if cur > 0 {
			cur--
		}
	case KeyDown:
		if cur < n-1 {
			cur++
		}
	case KeyPageUp:
		if cur >= vis {
			cur -= vis
		} else {
			cur = 0
		}
	case KeyPageDown:
		if cur < n-vis {
			cur += vis
		} else {
			cur = n - 1
		}
	}
	return cur
}

// visRows is how many table rows fit on screen, where available is the row
// budget left after the non-data furniture.
func visRows(available, n int) int {
	v := available
	if v < 1 {
		v = 1
	}
	if v > n {
		v = n
	}
	return v
}

func fixScroll(cur, top *int, vis, n int) {
	if *cur < *top {
		*top = *cur
	}
	if *cur >= *top+vis {
		*top = *cur - vis + 1
	}
}

// ---------------------------------------------------------------- rendering

func render(p Prompt, cols, rows, cur, top int, status string) {
	var b strings.Builder
	for i, l := range layout(p, cols, rows, cur, top, status) {
		b.WriteString("\x1b[")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(";1H\x1b[2K")
		b.WriteString(l)
	}
	b.WriteString("\x1b[J")
	_, _ = os.Stdout.WriteString(b.String())
}

func layout(p Prompt, cols, rows, cur, top int, status string) []string {
	// asciiChrome, not noneChrome: the picker draws the boxed table, and its
	// eight pipes are cells the columns do not get to use.
	w := computeWidths(p.Rows, cols, asciiChrome)
	// The table is rendered once over every row, so the column widths are set by
	// the whole result set rather than by the page on screen: scrolling does not
	// make the columns jiggle.
	rule, head, body := tableFrame(p.Rows, w)
	st := statusBlock(status, cols)
	vis := visRows(rows-chromeFor(status, cols), len(p.Rows))
	out := make([]string, 0, rows)
	out = append(out, cBold+clip(p.Title, cols)+cReset, "")
	out = append(out, cDim+rule+cReset)
	out = append(out, cBold+cCyan+head+cReset)
	out = append(out, cDim+rule+cReset)
	for i := top; i < top+vis && i < len(p.Rows); i++ {
		if i >= len(body) {
			break
		}
		if i == cur {
			body[i] = cRev + body[i] + cReset
		}
		out = append(out, body[i])
	}
	out = append(out, cDim+rule+cReset)
	out = append(out, "")
	out = append(out, st...)
	out = append(out, cDim+footer(cols, len(p.Rows), cur, p.Foot)+cReset)
	for len(out) < rows {
		out = append(out, "")
	}
	return out
}

// PrintTable writes the search result as plain text with no cursor movement, for
// a piped or redirected stdout. Same columns and alignment as the picker.
func PrintTable(w io.Writer, cols int, title string, rows []Row) {
	if cols <= 0 {
		cols = 80
	}
	if title != "" {
		fmt.Fprintf(w, "%s\n\n", title)
	}
	for _, ln := range renderTable(rows, computeWidths(rows, cols, asciiChrome), tw.StyleASCII) {
		fmt.Fprintln(w, ln)
	}
}

// statusLines is how many screen rows the status block occupies. It is the one
// source of truth: statusBlock renders exactly this many lines and chromeFor
// reserves exactly this many rows, so the two cannot drift. A status carrying
// an explicit newline always takes maxStatusLines, which keeps the row budget
// flat across a confirm-and-return round trip - otherwise the table would jump
// a row whenever the prompt appeared.
func statusLines(status string, cols int) int {
	n := len(wrapStatus(status, cols))
	if n < maxStatusLines && strings.Contains(status, "\n") {
		n = maxStatusLines
	}
	return n
}

// statusBlock renders the status text as one or two screen lines. Confirming a
// download needs two - the question, then the destination - and a long error
// message wraps to two as well, which is why the visible row count is derived
// from the status rather than taken from a constant. It pads out to
// statusLines so its length always equals the rows chromeFor reserved.
func statusBlock(status string, cols int) []string {
	n := statusLines(status, cols)
	out := make([]string, 0, n)
	for _, ln := range wrapStatus(status, cols) {
		out = append(out, statusLine(ln, cols))
	}
	for len(out) < n {
		out = append(out, "")
	}
	return out
}

// chromeFor is the row count the non-data furniture takes for one status text.
func chromeFor(status string, cols int) int {
	return fixedChrome + statusLines(status, cols)
}

// wrapStatus fits status into at most maxStatusLines display lines of cols cells
// each. Explicit newlines break first; longer text is wrapped, preferring a
// space so an English word does not split. Whatever does not fit is cut and the
// last line is marked with .., so a long diagnostic is still readable instead of
// vanishing behind a single clipped line.
func wrapStatus(status string, cols int) []string {
	lines := make([]string, 0, maxStatusLines)
	overflow := false
	for _, part := range strings.Split(status, "\n") {
		if overflow {
			break
		}
		if part == "" {
			if len(lines) < maxStatusLines {
				lines = append(lines, "")
			}
			continue
		}
		left := part
		for left != "" {
			if len(lines) == maxStatusLines {
				overflow = true
				break
			}
			line, rest := cutAt(left, cols)
			if rest != "" {
				lines = append(lines, clip(line, cols))
			} else {
				lines = append(lines, left)
			}
			left = rest
		}
		if left != "" {
			overflow = true
		}
	}
	if overflow && len(lines) > 0 {
		lines[len(lines)-1] = clip(lines[len(lines)-1]+"..", cols)
	}
	return lines
}

// cutAt splits s at a cell boundary: the longest prefix that fits width cells,
// then whatever is left. When the cut would land in the middle of an English
// word it backs up to the last space; Chinese has no spaces, so that is a no-op
// there. It adds no marker, because the caller decides whether the remainder
// deserves one.
func cutAt(s string, width int) (string, string) {
	if width <= 0 {
		return "", s
	}
	end := 0
	for end < len(s) {
		r, w := utf8.DecodeRuneInString(s[end:])
		wid := rw(r)
		if wid > 0 && disp(s[:end])+wid > width {
			break
		}
		end += w
	}
	if end == len(s) {
		return s, ""
	}
	fit, rest := s[:end], s[end:]
	if j := strings.LastIndexByte(fit, ' '); j > 0 {
		fit, rest = fit[:j], fit[j+1:]+rest
	}
	// A cut that lands on a space would otherwise indent the continuation line.
	rest = strings.TrimLeft(rest, " ")
	return fit, rest
}

func statusLine(status string, cols int) string {
	if strings.HasPrefix(status, "✗") {
		return cRed + clip(status, cols) + cReset
	}
	return cDim + clip(status, cols) + cReset
}

func footer(cols, n, cur int, foot string) string {
	s := fmt.Sprintf("%s   第 %d/%d 项   ↑↓ 选择   Enter 下载   q/Esc 退出",
		foot, cur+1, n)
	return clip(s, cols)
}

// ------------------------------------------------------------------ widths

// rw is the terminal display width of one rune in cells.
func rw(r rune) int {
	switch {
	case r == 0, r < 0x20, r >= 0x7f && r < 0xa0:
		return 0
	// Halfwidth katakana (U+FF61–U+FF9F) is inside unicode.Katakana but is one
	// cell wide, so it has to be tested before the wide table.
	case r >= 0xff61 && r <= 0xff9f:
		return 1
	case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r),
		r >= 0x3000 && r <= 0x303f, // 、。「」（）【】〔〕《》…
		r >= 0xff01 && r <= 0xff60, // fullwidth alphanumerics
		r >= 0xffe0 && r <= 0xffe6: // ¥ etc.
		return 2
	case r >= 0x1f300 && r <= 0x1faff, r >= 0x2600 && r <= 0x27bf:
		return 2
	}
	return 1
}

// disp is the visible width of s in cells. ANSI escape sequences occupy no
// cells on screen, but their bytes look like ordinary characters, so a
// sequence is skipped whole. CSI intermediate bytes are below 0x40 and the
// final byte is 0x40..0x7e; the "[" introducer sits in that range too, so
// it is consumed explicitly rather than letting the scan stop there.
func disp(s string) int {
	n, i := 0, 0
	for i < len(s) {
		if s[i] == 0x1b {
			i++
			if i >= len(s) {
				break
			}
			switch s[i] {
			case '[', 'O':
				i++
				for i < len(s) && s[i] < 0x40 {
					i++
				}
				if i < len(s) {
					i++
				}
			default: // two-character escapes such as \x1b7 or \x1bM
				i++
			}
			continue
		}
		r, w := utf8.DecodeRuneInString(s[i:])
		n += rw(r)
		i += w
	}
	return n
}

// clip shortens s to at most width cells, appending .. when something was cut.
// It never leaves a two-cell rune half-rendered at the boundary.
//
// The marker is two ASCII dots, not …, because tablewriter cannot measure an
// ellipsis: it drops U+2026 outright when the cell is exactly full, and sizes the
// cell one column too narrow when the cell is padded. Either way the body rows
// slid out from under the header. Dots are unambiguous width and tablewriter and
// rw agree on them, so the column widths stay put. … is still used where text is
// painted straight to the terminal and never goes through a table.
func clip(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if disp(s) <= width {
		return s
	}
	budget, tail := width, ""
	if width >= 3 {
		budget, tail = width-2, ".."
	}
	var b strings.Builder
	for _, r := range s {
		w := rw(r)
		if w > 0 && disp(b.String())+w > budget {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + tail
}
