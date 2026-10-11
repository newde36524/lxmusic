//go:build windows

package tui

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	enableVT     = 0x0004
	genericRW    = 0x80000000 | 0x40000000 | 0x00100000
	openExisting = 3

	// The standard handles are negative decimal sentinels. Written in
	// two's-complement form rather than as a negative constant, because Go
	// refuses to convert a negative constant to an unsigned type and -11
	// would not fit a 32-bit uintptr as a literal either.
	stdOut = ^uintptr(0) - 10 // STD_OUTPUT_HANDLE == -11
	stdIn  = ^uintptr(0) - 9  // STD_INPUT_HANDLE == -10

	keyEvent = 1

	vkUp     = 0x26
	vkDown   = 0x28
	vkPrior  = 0x21
	vkNext   = 0x22
	vkHome   = 0x24
	vkEnd    = 0x23
	vkReturn = 0x0d
	vkSpace  = 0x20
	vkEscape = 0x1b

	// dwControlKeyState modifier bits. Kept separate from the enhanced-key
	// flag (0x0100) so an enhanced numpad key is not mistaken for a modifier.
	modShift = 0x0010 | 0x0200 | 0x0400
	modCtrl  = 0x0004 | 0x0008
	modAlt   = 0x0001 | 0x0002

	// INPUT_RECORD field offsets, verified on this platform by writing a
	// synthetic key event with WriteConsoleInputW and reading it back. A
	// 64-byte buffer is safely above every documented record size.
	offEventType = 0
	offVK        = 10
	offUChar     = 14
	offKeyState  = 16

	// keyEchoWindow is how long a decoded key stays recent enough to swallow its
	// own echo. Wide on purpose: it only has to exceed the delay between a
	// key-DOWN and the key-UP that follows it, and it never has to be narrow
	// enough to let a held arrow key auto-repeat. A 20-row list does not need
	// auto-repeat.
	keyEchoWindow = 100 * time.Millisecond
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	pGetStdHandle      = kernel32.NewProc("GetStdHandle")
	pCreateFileW       = kernel32.NewProc("CreateFileW")
	pCloseHandle       = kernel32.NewProc("CloseHandle")
	pGetConsoleMode    = kernel32.NewProc("GetConsoleMode")
	pSetConsoleMode    = kernel32.NewProc("SetConsoleMode")
	pReadConsoleInputW = kernel32.NewProc("ReadConsoleInputW")
	pScreenBufferInfo  = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

var (
	outHandle uintptr
	inHandle  uintptr
	createdIn bool
	oldMode   uint32
	haveMode  bool
)

// keySig is the identity of a key for echo detection: the virtual key code and
// the character it produced. Two records with the same identity are the same
// physical press until told otherwise.
type keySig struct {
	vk uint16
	ch uint16
}

// keyPress is the last record that reached the UI, with the clock reading that
// decided it was a real press. Only nextKey's goroutine touches it, so it needs
// no lock.
type keyPress struct {
	sig keySig
	at  time.Time
}

var lastPress keyPress

func stdHandle(n uintptr) uintptr {
	r, _, _ := pGetStdHandle.Call(n)
	return r
}

func isConsole(h uintptr) bool {
	var mode uint32
	r, _, _ := pGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	return r != 0
}

// isTerminal is true only when both standard streams are real consoles.
// Redirected or piped output must never receive escape sequences.
func isTerminal() bool {
	out, in := stdHandle(stdOut), stdHandle(stdIn)
	return out != 0 && isConsole(out) && in != 0 && isConsole(in)
}

// enable turns on virtual terminal processing and opens a console input handle.
// STD_INPUT_HANDLE is unusable in a sandboxed process, so fall back to CONIN$,
// tracking the handle so disable can close it.
func enable() error {
	outHandle = stdHandle(stdOut)
	if outHandle == 0 {
		return os.ErrInvalid
	}
	var mode uint32
	if r, _, _ := pGetConsoleMode.Call(outHandle, uintptr(unsafe.Pointer(&mode))); r == 0 {
		return os.ErrInvalid
	}
	if r, _, _ := pSetConsoleMode.Call(outHandle, uintptr(mode|enableVT)); r == 0 {
		return os.ErrInvalid
	}
	oldMode, haveMode = mode, true

	h := stdHandle(stdIn)
	if h != 0 && isConsole(h) {
		inHandle = h
	} else {
		name, _ := syscall.UTF16PtrFromString("CONIN$")
		inHandle, _, _ = pCreateFileW.Call(uintptr(unsafe.Pointer(name)),
			uintptr(genericRW), 0, 0, openExisting, 0, 0)
		if inHandle == 0 {
			return os.ErrInvalid
		}
		createdIn = true
	}

	_, _ = os.Stdout.WriteString("\x1b[?1049h\x1b[2J\x1b[H")
	return nil
}

func disable() {
	_, _ = os.Stdout.WriteString("\x1b[?1049l")
	if haveMode {
		pSetConsoleMode.Call(outHandle, uintptr(oldMode))
	}
	if createdIn {
		pCloseHandle.Call(inHandle)
	}
}

// CONSOLE_SCREEN_BUFFER_INFO layout, as documented offsets:
//
//	COORD  dwSize                 0   X@0   Y@2
//	COORD  dwCursorPosition       4
//	WORD   attributes             8
//	SMALL_RECT srWindow           10  Left@10 Top@12 Right@14 Bottom@16
//	COORD  dwMaximumWindowSize   18
func termSize() (int, int) {
	h := stdHandle(stdOut)
	if h == 0 {
		return 80, 24
	}
	buf := make([]byte, 64)
	if r, _, _ := pScreenBufferInfo.Call(h, uintptr(unsafe.Pointer(&buf[0]))); r == 0 {
		return 80, 24
	}
	// The visible window, not dwSize: dwSize is the scroll-back buffer, which
	// under a pty can be wider than the pane being drawn into and would send the
	// table past the right edge. The window rect is the pane.
	left, top, right, bot := le16(buf, 10), le16(buf, 12), le16(buf, 14), le16(buf, 16)
	cols := int(right-left) + 1
	rows := int(bot-top) + 1
	if cols < 20 {
		cols = int(le16(buf, 0)) // fall back to the buffer size
	}
	if cols < 20 {
		cols = 80
	}
	if rows < 8 {
		rows = 24
	}
	return cols, rows
}

func nextKey() (Key, error) {
	if inHandle == 0 {
		return KeyNone, os.ErrInvalid
	}
	buf := make([]byte, 64)
	var n uint32
	for {
		r, _, err := pReadConsoleInputW.Call(inHandle, uintptr(unsafe.Pointer(&buf[0])), 1,
			uintptr(unsafe.Pointer(&n)))
		if r == 0 {
			return KeyNone, err
		}
		if n == 0 {
			continue
		}
		if k := recordKey(buf, time.Now()); k != KeyNone {
			return k, nil
		}
	}
}

// recordKey decodes one INPUT_RECORD, or reports KeyNone for a record that
// should not move anything. now is passed in rather than read, so the echo
// logic can be tested against a fixed clock.
//
// A physical key press produces two records: a key-DOWN and, a few hundred
// microseconds later, a key-UP carrying the same virtual key code and the same
// character. Both are event type KEY_EVENT, so the event type cannot tell them
// apart, and the two halves used to reach the UI as two presses - which moved
// the selection two rows per arrow tap and, worse, answered the download
// confirmation for itself, so pressing Enter never got to show the prompt.
//
// They are distinguished by timing, not by any field in the record. The obvious
// field is bKeyDown, but there is no offset for it that survives contact with
// the offsets that do decode correctly: the records prove the filter at offset
// 8 was letting key-ups through, so byte 8 is not bKeyDown - and in every
// documented layout of KEY_EVENT_RECORD, offset 8 holds wRepeatCount or
// wVirtualKeyCode, both of which are nonzero for both halves of a press.
// Nothing in the record ever said a key was up, so there was nothing to read.
func recordKey(buf []byte, now time.Time) Key {
	if le16(buf, offEventType) != keyEvent {
		return KeyNone
	}
	sig := keySig{vk: le16(buf, offVK), ch: le16(buf, offUChar)}
	if !lastPress.at.IsZero() && sig == lastPress.sig &&
		now.Sub(lastPress.at) < keyEchoWindow {
		return KeyNone
	}
	lastPress = keyPress{sig: sig, at: now}
	return winKey(sig.vk, sig.ch, le32(buf, offKeyState))
}

// winKey normalises one key event. A held modifier is dropped for the movement
// keys so Ctrl+arrow and Alt+arrow never drive the selection.
func winKey(vk, ch uint16, state uint32) Key {
	if ch == 3 || ch == 4 { // Ctrl+C / Ctrl+D
		return KeyQuit
	}
	anyMod := state&(modShift|modCtrl|modAlt) != 0
	switch vk {
	case vkUp:
		if !anyMod {
			return KeyUp
		}
	case vkDown:
		if !anyMod {
			return KeyDown
		}
	case vkPrior, vkHome:
		if !anyMod {
			return KeyPageUp
		}
	case vkNext, vkEnd:
		if !anyMod {
			return KeyPageDown
		}
	case vkReturn, vkSpace:
		return KeyEnter
	case vkEscape:
		return KeyQuit
	}
	switch ch {
	case 'q', 'Q':
		return KeyQuit
	case 'k', 'K':
		return KeyUp
	case 'j', 'J':
		return KeyDown
	case 'g', 'G':
		return KeyPageUp
	case 'y', 'Y':
		return KeyYes
	case 'n', 'N':
		return KeyNo
	}
	return KeyNone
}

func le16(b []byte, off int) uint16 { return uint16(b[off]) | uint16(b[off+1])<<8 }

func le32(b []byte, off int) uint32 {
	return uint32(b[off]) | uint32(b[off+1])<<8 |
		uint32(b[off+2])<<16 | uint32(b[off+3])<<24
}
