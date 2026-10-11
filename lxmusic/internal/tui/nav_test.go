package tui

import (
	"strconv"
	"testing"
)

// TestMoveSelection pins the bounds of every navigation key. The page keys are
// the ones worth pinning: with a list longer than a screen they must land on a
// page boundary and stop exactly at the last row, and none of them may step
// outside [0, n-1] - not at either end, and not when the screen shows more rows
// than there are.
func TestMoveSelection(t *testing.T) {
	tests := []struct {
		key  Key
		cur  int
		vis  int
		n    int
		want int
	}{
		// Arrows: one row, clamped at both ends.
		{KeyUp, 0, 20, 50, 0},
		{KeyUp, 1, 20, 50, 0},
		{KeyUp, 25, 20, 50, 24},
		{KeyDown, 0, 20, 50, 1},
		{KeyDown, 48, 20, 50, 49},
		{KeyDown, 49, 20, 50, 49},
		// PgUp: back a page, or to the top if a page would not fit.
		{KeyPageUp, 0, 20, 50, 0},
		{KeyPageUp, 19, 20, 50, 0},
		{KeyPageUp, 20, 20, 50, 0},
		{KeyPageUp, 25, 20, 50, 5},
		{KeyPageUp, 40, 20, 50, 20},
		{KeyPageUp, 49, 20, 50, 29},
		// PgDn: forward a page, or to the last row.
		{KeyPageDown, 0, 20, 50, 20},
		{KeyPageDown, 10, 20, 50, 30},
		{KeyPageDown, 25, 20, 50, 45},
		{KeyPageDown, 30, 20, 50, 49},
		{KeyPageDown, 49, 20, 50, 49},
		// A short list: nothing to page through, everything clamps to the ends.
		{KeyPageUp, 3, 20, 10, 0},
		{KeyPageDown, 0, 20, 10, 9},
		{KeyPageDown, 5, 20, 10, 9},
		// vis larger than the list: n-vis is negative, so PgDn must not underflow.
		{KeyPageDown, 0, 100, 5, 4},
		{KeyPageUp, 0, 100, 5, 0},
		// Home and End are wired to PgUp and PgDn, which is what puts them on the
		// first and last row.
		{KeyPageUp, 49, 20, 50, 29},
		{KeyPageDown, 0, 20, 50, 20},
		// One row only.
		{KeyPageDown, 0, 20, 1, 0},
		{KeyUp, 0, 20, 1, 0},
		{KeyDown, 0, 20, 1, 0},
		// An empty list and a non-navigation key.
		{KeyDown, 0, 20, 0, 0},
		{KeyPageDown, 5, 20, 0, 0},
		{KeyEnter, 7, 20, 50, 7},
		{KeyQuit, 7, 20, 50, 7},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(int(tt.key))+"@"+strconv.Itoa(tt.cur), func(t *testing.T) {
			if got := moveSelection(tt.key, tt.cur, tt.vis, tt.n); got != tt.want {
				t.Fatalf("moveSelection(%v, %d, vis=%d, n=%d) = %d, want %d",
					tt.key, tt.cur, tt.vis, tt.n, got, tt.want)
			}
		})
	}
}

// TestMoveSelectionInBounds sweeps keys and positions and asserts the
// selection never leaves [0, n-1], which is the property that actually matters
// to the picker: an out-of-range index is a panic in layout, not a bad frame.
func TestMoveSelectionInBounds(t *testing.T) {
	keys := []Key{KeyUp, KeyDown, KeyPageUp, KeyPageDown}
	for n := 1; n <= 60; n++ {
		for vis := 1; vis <= 60; vis++ {
			for cur := 0; cur < n; cur++ {
				for _, k := range keys {
					if got := moveSelection(k, cur, vis, n); got < 0 || got >= n {
						t.Fatalf("moveSelection(%v, %d, vis=%d, n=%d) = %d, out of range",
							k, cur, vis, n, got)
					}
				}
			}
		}
	}
}
