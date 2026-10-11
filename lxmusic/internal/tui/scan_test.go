package tui

import "testing"

// TestScanKeys feeds the byte streams a real terminal produces and checks the
// normalised result. The parser is pure byte logic with no build tag, so this
// suite runs on whatever host runs `go test` — no terminal attached.
func TestScanKeys(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Key
		rest string
	}{
		{"up", "\x1b[A", KeyUp, ""},
		{"down", "\x1b[B", KeyDown, ""},
		{"pageup", "\x1b[5~", KeyPageUp, ""},
		{"pagedown", "\x1b[6~", KeyPageDown, ""},
		{"home", "\x1b[1~", KeyPageUp, ""},
		{"end", "\x1b[4~", KeyPageDown, ""},
		{"pageup-alt", "\x1b[7~", KeyPageUp, ""},
		{"tmux-up", "\x1bOA", KeyUp, ""},
		{"tmux-down", "\x1bOB", KeyDown, ""},
		{"enter-cr", "\r", KeyEnter, ""},
		{"enter-lf", "\n", KeyEnter, ""},
		{"ctrl-c", "\x03", KeyQuit, ""},
		{"ctrl-d", "\x04", KeyQuit, ""},
		{"q", "q", KeyQuit, ""},
		{"j-up", "k", KeyUp, ""},
		{"j-down", "j", KeyDown, ""},
		{"j-top", "g", KeyPageUp, ""},
		{"yes-lower", "y", KeyYes, ""},
		{"yes-upper", "Y", KeyYes, ""},
		{"no-lower", "n", KeyNo, ""},
		{"no-upper", "N", KeyNo, ""},
		{"ignored", "x", KeyNone, ""},
		{"up-then-x", "\x1b[Ax", KeyUp, "x"},
	}
	for _, c := range cases {
		got, rest, ok := scan([]byte(c.in))
		if !ok {
			t.Errorf("%s: scan(%q) incomplete, want a key", c.name, c.in)
			continue
		}
		if got != c.want {
			t.Errorf("%s: scan(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
		if gotRest := string(rest); gotRest != c.rest {
			t.Errorf("%s: leftover %q, want %q", c.name, gotRest, c.rest)
		}
	}
}

// TestScanIgnorable covers the other branch: a buffer that holds only noise is
// a verdict, so it must not report itself as truncated. Before the fix scan("x")
// returned ok=false, which meant nextKey would sit blocked on noise forever.
func TestScanIgnorable(t *testing.T) {
	for _, in := range []string{"", "x", "xxx", "\x1b[Axy"} {
		_, rest, ok := scan([]byte(in))
		if !ok {
			t.Errorf("scan(%q) reported incomplete, want a verdict", in)
			continue
		}
		if in == "\x1b[Axy" {
			if got, want := string(rest), "xy"; got != want {
				t.Errorf("scan(%q) leftover %q, want %q", in, got, want)
			}
		} else if len(rest) != 0 {
			t.Errorf("scan(%q) leftover %q, want empty", in, rest)
		}
	}
}

// TestScanIncomplete confirms a multi-byte escape split across two reads is
// completed rather than being decoded as an ordinary character.
func TestScanIncomplete(t *testing.T) {
	for _, part := range []string{"\x1b", "\x1b[", "\x1b[5"} {
		if _, _, ok := scan([]byte(part)); ok {
			t.Errorf("scan(%q) reported complete for an incomplete sequence", part)
		}
	}
	// Two reads that together form one page-up.
	_, rest, ok := scan([]byte("\x1b[5"))
	if ok {
		t.Fatalf("scan(\\x1b[5) should be incomplete")
	}
	got, _, ok := scan(append(rest, '~'))
	if !ok || got != KeyPageUp {
		t.Fatalf("split sequence = %v,%v, want KeyPageUp,true", got, ok)
	}
}
