//go:build windows

package tui

import (
	"testing"
	"time"
)

// These are ABI facts rather than style choices. A wrong sentinel fails in
// one place and a wrong offset decodes the wrong field, so pin both.
func TestHandleSentinels(t *testing.T) {
	// The sentinels are stored unsigned, so int(stdOut) is a compile-time
	// overflow: ^uintptr(0)-10 is 18446744073709551605, above int's range.
	// Widen through a variable to get the runtime conversion, which is what
	// the syscall layer actually does and which gives the documented value.
	var out, in uintptr = stdOut, stdIn
	if got := int(out); got != -11 {
		t.Errorf("int(stdOut) = %d, want -11", got)
	}
	if got := int(in); got != -10 {
		t.Errorf("int(stdIn) = %d, want -10", got)
	}
}

func TestInputRecordOffsets(t *testing.T) {
	if offVK != 10 || offUChar != 14 || offKeyState != 16 {
		t.Errorf("INPUT_RECORD offsets drifted: VK=%d UChar=%d State=%d",
			offVK, offUChar, offKeyState)
	}
}

// rec builds one synthetic INPUT_RECORD at the offsets verified in
// probe/layout2. It fills only the fields recordKey reads, because the key-up
// half of a press differs from the key-down half by nothing but the clock.
func rec(vk, ch uint16) []byte {
	b := make([]byte, 64)
	b[offEventType] = keyEvent
	b[offVK] = byte(vk)
	b[offVK+1] = byte(vk >> 8)
	b[offUChar] = byte(ch)
	b[offUChar+1] = byte(ch >> 8)
	return b
}

var testNow = time.Unix(1_700_000_000, 0)

func resetKeyHistory() { lastPress = keyPress{} }

// TestRecordKeyFiltersKeyUp is the regression test for the two real bugs the
// key-up records caused: the selection moving two rows per arrow tap, and the
// download confirmation answering itself so Enter never showed the prompt.
// Each press delivers its down half and, microseconds later, an up half with
// identical virtual key code and character.
func TestRecordKeyFiltersKeyUp(t *testing.T) {
	for _, c := range []struct {
		name string
		vk   uint16
		ch   uint16
		want Key
	}{
		{"arrow-down", vkDown, 0, KeyDown},
		{"arrow-up", vkUp, 0, KeyUp},
		{"enter", vkReturn, 0, KeyEnter},
		{"yes", 0, 'y', KeyYes},
		{"no", 0, 'n', KeyNo},
		{"pagedown", vkNext, 0, KeyPageDown},
		{"quit", 0, 'q', KeyQuit},
	} {
		b := rec(c.vk, c.ch)
		resetKeyHistory()
		if got := recordKey(b, testNow); got != c.want {
			t.Errorf("%s down: recordKey = %v, want %v", c.name, got, c.want)
		}
		// The up half of the same press arrives with the same codes.
		if got := recordKey(b, testNow.Add(300*time.Microsecond)); got != KeyNone {
			t.Errorf("%s up: recordKey = %v, want KeyNone", c.name, got)
		}
		// And the next press, well outside the echo window, must register.
		if got := recordKey(b, testNow.Add(500*time.Millisecond)); got != c.want {
			t.Errorf("%s repeat: recordKey = %v, want %v", c.name, got, c.want)
		}
	}
	resetKeyHistory()
	if got := recordKey(make([]byte, 64), testNow); got != KeyNone {
		t.Errorf("all-zero record decoded as %v, want KeyNone", got)
	}
	// A character event (type 2) must never decode as a key no matter what the
	// bytes say, and must not poison the history behind it.
	ce := rec(vkDown, 0)
	ce[offEventType] = 2
	if got := recordKey(ce, testNow); got != KeyNone {
		t.Errorf("char event decoded as %v, want KeyNone", got)
	}
	if got := recordKey(rec(vkDown, 0), testNow.Add(time.Millisecond)); got != KeyDown {
		t.Errorf("a char event poisoned the history: got %v, want KeyDown", got)
	}
}

// TestRecordKeyEchoWindow pins both edges of the echo window: an identical key
// inside it is the up half of a press and must be swallowed, outside it is a
// deliberate second press and must not be.
func TestRecordKeyEchoWindow(t *testing.T) {
	for _, d := range []time.Duration{1 * time.Microsecond, time.Millisecond, 50 * time.Millisecond} {
		resetKeyHistory()
		if got := recordKey(rec(vkDown, 0), testNow); got != KeyDown {
			t.Fatalf("delay %v: down = %v", d, got)
		}
		if got := recordKey(rec(vkDown, 0), testNow.Add(d)); got != KeyNone {
			t.Errorf("delay %v inside the window decoded as %v, want KeyNone", d, got)
		}
	}
	resetKeyHistory()
	_ = recordKey(rec(vkDown, 0), testNow)
	if got := recordKey(rec(vkDown, 0), testNow.Add(keyEchoWindow+time.Millisecond)); got != KeyDown {
		t.Errorf("outside the window = %v, want KeyDown", got)
	}
	// A different key is never an echo of another, however close it lands.
	resetKeyHistory()
	_ = recordKey(rec(vkUp, 0), testNow)
	if got := recordKey(rec(vkDown, 0), testNow.Add(time.Microsecond)); got != KeyDown {
		t.Errorf("a different key was swallowed: %v, want KeyDown", got)
	}
}
