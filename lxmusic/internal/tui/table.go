package tui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

const cellPad = 2 // tablewriter's one-cell padding on either side of the text

// headers is the picker table's header row. The size unit sits in the cells
// rather than in this label: "MB" in parentheses would read as 文件大小(MB), but
// tablewriter measures ASCII punctuation with East Asian width rules, where
// "(" and ")" count two cells, and the label would then not fit its column.
var headers = []string{"序号", "平台名称", "音乐名称", "时长", "作者", "音质", "文件大小"}

// asciiChrome is the width of StyleASCII's border characters: a pipe at each
// end plus one between every pair of columns, so len(headers)+1. Counted rather
// than written as a literal, because a hardcoded 7 left the newest column
// squeezed against the right edge of an 80-column table.
var asciiChrome = len(headers) + 1

// noneChrome is how much column width StyleNone's border characters take:
// nothing, so the columns fill the terminal exactly.
const noneChrome = 0

// aligns is the per-column alignment: the counters right, the text left.
var aligns = tw.Alignment{
	tw.AlignRight,
	tw.AlignLeft,
	tw.AlignLeft,
	tw.AlignRight,
	tw.AlignLeft,
	tw.AlignLeft,
	tw.AlignRight,
}

// widths holds the declared column widths for one terminal size, in header order.
type widths struct {
	index, platform, name, duration, artist, quality, size int
}

func (w widths) all() []int {
	return []int{w.index, w.platform, w.name, w.duration, w.artist, w.quality, w.size}
}

func (w *widths) set(v []int) {
	w.index, w.platform, w.name, w.duration, w.artist, w.quality, w.size =
		v[0], v[1], v[2], v[3], v[4], v[5], v[6]
}

func (w widths) total() int { return sumOf(w.all()) }

// headerFloors is the narrowest each column may be squeezed to: the longer of its
// own caption plus padding, or the width its data actually needs.
func headerFloors() []int {
	f := make([]int, len(headers))
	for i, h := range headers {
		if v := disp(h) + cellPad; v > valueFloors[i] {
			f[i] = v
		} else {
			f[i] = valueFloors[i]
		}
	}
	return f
}

// valueFloors is how wide each column must be for its data to stay true,
// whatever the caption says. Below these the columns lie rather than simply
// elide: 时长 at 6 cells shows "04.." where a duration is always "MM:SS" at 5,
// 文件大小 at 8 shows "953.15" without its unit, and 音质 at 6 turns "flac24bit"
// into "flac.." - indistinguishable from "flac", and the list is sorted by it.
// Platform, name and artist stay at their caption width: a cut title is lossy
// but still names the song, whereas a cut number is just a wrong number.
var valueFloors = []int{4, 6, 8, 7, 6, 11, 10}

// minCols is the narrowest column area this table is drawn in: the sum of the
// header floors. At that width every column still shows its caption, and only
// the cell text is trimmed.
var minCols = func() int {
	t := 0
	for _, n := range headerFloors() {
		t += n
	}
	return t
}()

// pickerMin is the narrowest terminal the picker is drawn at: minCols plus the
// pipes of the box it draws the table inside. Widths are computed against
// asciiChrome, so a terminal at minCols alone would put the table eight cells
// past its right edge.
var pickerMin = minCols + asciiChrome

// sizeCell renders a byte count in MB, the unit the column carries on every row
// so the header stays clear of ambiguous-width punctuation. A zero means the
// platform announced no size - not a zero-byte file - so it shows as "-" rather
// than 0.00MB, which would read as a real and tiny file.
func sizeCell(bytes int64) string {
	if bytes <= 0 {
		return "-"
	}
	return humanBytes(bytes)
}

// cellText is one row's seven cells with no clipping applied, so the width maths
// can measure them as they really are.
func cellText(r Row) []string {
	q := r.Quality
	if q == "" {
		q = "-"
	}
	return []string{
		fmt.Sprintf("%d", r.Index),
		r.Platform, r.Name, r.Duration, r.Artist, q, sizeCell(r.Size),
	}
}

// columnWidths sizes every column to fit its content: its caption and its widest
// value, whichever is longer, padding included. A table rendered at these widths
// hides nothing, which is the point of measuring rather than flooring.
func columnWidths(rows []Row) widths {
	all := make([][]string, 0, len(rows)+1)
	all = append(all, headers)
	for _, r := range rows {
		all = append(all, cellText(r))
	}
	w := widths{}
	for i, cur := range headerFloors() {
		if h := disp(all[0][i]) + cellPad; h > cur {
			cur = h
		}
		for _, c := range all[1:] {
			if v := disp(c[i]) + cellPad; v > cur {
				cur = v
			}
		}
		switch i {
		case 0:
			w.index = cur
		case 1:
			w.platform = cur
		case 2:
			w.name = cur
		case 3:
			w.duration = cur
		case 4:
			w.artist = cur
		case 5:
			w.quality = cur
		case 6:
			w.size = cur
		}
	}
	return w
}

// shrinkGroups is the order a column's spare width is surrendered when the
// terminal is too narrow for the content. The free-form text goes first: a title
// trimmed to "晴天 (女声独唱…" still says which song it is, but "flac24b" or
// "93.71" says neither which tier nor which size. Platform is next - short and
// fixed, so a little trimming still reads. The measured fields go last, since a
// cut value is just a wrong number.
var shrinkGroups = [][]int{{2, 4}, {1}, {3, 5, 6, 0}}

// computeWidths fits the table into cols cells. Content widths come first, then
// any slack is given to the song title and the artist, and only if it still does
// not fit is anything trimmed - never below a column's own caption.
func computeWidths(rows []Row, cols, chrome int) widths {
	if chrome < 0 {
		chrome = 0
	}
	w := columnWidths(rows)
	if total := chrome + w.total(); total <= cols {
		// The surplus is pure padding: the two free-form text columns take it,
		// the title a little more than the artist so a long title still reads as
		// the main field.
		for slack := cols - total; slack > 0; {
			if slack >= 3 {
				w.name += 2
				w.artist++
				slack -= 3
			} else {
				w.name += slack
				slack = 0
			}
		}
		return w
	}
	w.set(fitShrink(w.all(), headerFloors(), cols-chrome))
	handOut(&w, cols-chrome, columnWidths(rows).all())
	return w
}

// handOut spreads the width a shrink left behind back to the free-text columns,
// up to each one's measured width.
//
// fitShrink only ever takes, so a shrink that lands on the floors strands the
// surplus: the measured columns have no room left to give, the loop breaks, and
// the table sits at its minimum width in an 80-column terminal - 68 of the 80
// used, with the title and artist cells clipped to "晴天 (.." and "告..". The
// title gets two thirds of each round, matching the slack rule above, and
// nothing is pushed past what the content actually needs, since width beyond the
// longest value is decoration rather than information.
func handOut(w *widths, budget int, ideal []int) {
	for {
		left := budget - w.total()
		if left <= 0 {
			return
		}
		v := w.all()
		used := 0
		for _, s := range []struct {
			col  int
			room int
		}{
			{2, 2}, {4, 1},
		} {
			if left-used <= 0 || v[s.col] >= ideal[s.col] {
				continue
			}
			n := s.room
			if n > left-used {
				n = left - used
			}
			if n > ideal[s.col]-v[s.col] {
				n = ideal[s.col] - v[s.col]
			}
			v[s.col] += n
			used += n
		}
		w.set(v)
		if used == 0 {
			return
		}
	}
}

// fitShrink trims the widths until they fit budget, surrendering whole groups of
// columns before moving to the next. Within a group the trim is proportional to
// each column's slack, so nothing is reduced to a stub while its neighbours stay
// roomy. Nothing is taken below its caption width; if even that is not enough
// the content genuinely does not fit the terminal.
func fitShrink(v []int, floors []int, budget int) []int {
	for _, group := range shrinkGroups {
		for over := sumOf(v) - budget; over > 0; {
			room := 0
			for _, i := range group {
				if v[i] > floors[i] {
					room += v[i] - floors[i]
				}
			}
			if room == 0 {
				break
			}
			if room < over {
				for _, i := range group {
					v[i] = floors[i]
				}
				break
			}
			moved := false
			for _, i := range group {
				e := v[i] - floors[i]
				if e <= 0 {
					continue
				}
				take := int(float64(over)*float64(e)/float64(room) + 0.5)
				if take < 1 {
					take = 1
				}
				if take > e {
					take = e
				}
				v[i] -= take
				moved = true
			}
			if !moved {
				break
			}
		}
	}
	return v
}

func sumOf(v []int) int {
	t := 0
	for _, n := range v {
		t += n
	}
	return t
}

// cells is one row's seven column values, clipped so none of them wraps onto a
// second line: tablewriter splits an overflowing cell instead of eliding it, and
// a wrapped row would break the one-line-per-row mapping the picker depends on.
func cells(r Row, w widths) []string {
	v := cellText(r)
	out := make([]string, len(v))
	for i, cw := range w.all() {
		out[i] = padCell(v[i], cw-cellPad, aligns[i])
	}
	return out
}

// padCell clips s to the column's content width and then pads it back out to
// that width, justifying as the column asks.
//
// Clipping alone does not hold the table together. tablewriter sizes a column
// from the widest cell it sees, using its own East Asian width rules, and it is
// not consistent: for "04:…" it honoured a declared width of 6, for "周…" of the
// same content width it used content+2 = 5 and dropped the column one cell
// narrower. Padding makes every cell exactly cw-cellPad cells, so content+2
// always equals the declared width and there is nothing left for it to decide.
func padCell(s string, w int, align tw.Align) string {
	s = clip(s, w)
	d := disp(s)
	switch align {
	case tw.AlignRight:
		return strings.Repeat(" ", w-d) + s
	case tw.AlignCenter:
		left := (w - d) / 2
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-d-left)
	}
	return s + strings.Repeat(" ", w-d)
}

// renderTable is the one call both PrintTable and the picker go through, so the
// plain output and the interactive table cannot drift apart on column widths or
// alignment. It returns the finished table, one line per row, with no cursor
// movement in it.
func renderTable(rows []Row, w widths, style tw.BorderStyle) []string {
	m := tw.Mapper[int, int]{}
	for i, cw := range w.all() {
		m[i] = cw
	}
	// The header is padded to the same width as the body cells. Left to its own
	// content a caption would be wider than a column, tablewriter would widen
	// that column to keep the caption intact, and the body rows would land one
	// cell short of the header again.
	head := make([]string, len(headers))
	for i, cw := range w.all() {
		head[i] = padCell(headers[i], cw-cellPad, tw.AlignCenter)
	}
	var b bytes.Buffer
	t := tablewriter.NewTable(&b,
		tablewriter.WithHeader(head),
		tablewriter.WithHeaderAlignment(tw.AlignCenter),
		tablewriter.WithAlignment(aligns),
		tablewriter.WithEastAsian(tw.Success),
		tablewriter.WithSymbols(tw.NewSymbols(style)),
		tablewriter.WithColumnWidths(m),
	)
	for _, r := range rows {
		if err := t.Append(cells(r, w)); err != nil {
			return nil
		}
	}
	if err := t.Render(); err != nil {
		return nil
	}
	return strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
}

// tableFrame renders the table as a boxed whole and splits it back into its
// pieces: the rule that frames the header, the header row, and one row per
// result. Everything keeps its borders.
//
// The borders are not decoration here - they are what forces alignment. The
// picker used to render StyleNone, which does not honour the declared column
// widths the way StyleASCII does: the header row came out a couple of cells
// wider than the body rows, so every caption sat to the right of the value it
// labelled and the whole table read as squashed. StyleASCII's rule is exactly
// sum(widths)+asciiChrome cells wide, so the header and the body are aligned by
// construction. The rule is emitted once and reused for the top, the header
// underline and the bottom, which is how StyleASCII draws it anyway.
func tableFrame(rows []Row, w widths) (rule, head string, body []string) {
	for _, ln := range renderTable(rows, w, tw.StyleASCII) {
		if strings.Contains(ln, "|") {
			if head == "" {
				head = ln
				continue
			}
			body = append(body, ln)
		} else if rule == "" {
			rule = ln
		}
	}
	return rule, head, body
}
