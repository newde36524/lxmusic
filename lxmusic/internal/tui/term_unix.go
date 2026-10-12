//go:build linux || darwin || freebsd || dragonfly || netbsd || openbsd

package tui

import (
	"os"
	"syscall"
	"unsafe"
)

// winsz is struct winsize: four uint16, on every unix.
type winsz struct{ Row, Col, Xpixel, Ypixel uint16 }

// TIOCGWINSZ keeps the same encoding on every unix, so it is looked up once
// here rather than per platform. tcgets and tcssets differ between kernels and
// come from the term_* files.
const winsize = syscall.TIOCGWINSZ

var saved *syscall.Termios

// ioctl wraps one ioctl and keeps its errno as an Errno, not an error.
// syscall.Syscall hands back a typed errno even on success, and a typed zero is
// a non-nil error: `err != nil` is true and reports "errno 0". So the result is
// returned as Errno for the caller to compare against zero, never against nil -
// translating it here, once, is what keeps the two call styles from
// coexisting. Before that, isTerminal() thought every real terminal was a
// pipe, so `lxmusic search` printed its table and exited instead of opening the
// picker.
func ioctl(trap uintptr, fd int, p unsafe.Pointer) syscall.Errno {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), trap, uintptr(p))
	return err
}

// isTerminal is true only when both standard streams are real terminals.
// Redirected or piped output must never receive escape sequences, and a
// redirected stdin is not interactive even if stdout is a terminal - without
// that second check `lxmusic search ... </dev/null` would enter the picker
// and then fail to take over the terminal.
func isTerminal() bool {
	var t syscall.Termios
	if err := ioctl(tcgets, int(os.Stdout.Fd()), unsafe.Pointer(&t)); err != 0 {
		return false
	}
	return ioctl(tcgets, int(os.Stdin.Fd()), unsafe.Pointer(&t)) == 0
}

// enable drops the terminal into raw mode so keystrokes arrive immediately.
// Only ECHO, ICANON and ISIG are cleared - those are the three this renderer
// needs - and their values come from the platform headers rather than being
// hand-composed: ICANON and ISIG sit at 0x100 and 0x80 on every BSD, and at
// 0x2 and 0x1 on Linux. Hardcoding one kernel's pair cleared the wrong bits on
// the others, so ICANON survived on the BSDs and the terminal kept input
// canonical, which made a read block until the user hit Enter. OPOST and IXON
// move too, but nothing here depends on either, because the renderer never
// writes a newline.
//
// VMIN is 0 and VTIME is 1 (100 ms): a read returns at once when a byte is
// there, but times out instead of blocking forever. That timeout is what lets
// nextKey tell a lone ESC from the start of a sequence that was split across
// reads, without adding latency to every ordinary keypress.
func enable() error {
	fd := int(os.Stdin.Fd())
	var old syscall.Termios
	if err := ioctl(tcgets, fd, unsafe.Pointer(&old)); err != 0 {
		return err
	}
	raw := old
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	raw.Cc[syscall.VMIN] = 0
	raw.Cc[syscall.VTIME] = 1
	if err := ioctl(tcssets, fd, unsafe.Pointer(&raw)); err != 0 {
		return err
	}
	saved = &old
	_, _ = os.Stdout.WriteString("\x1b[?1049h\x1b[2J\x1b[H\x1b[?25l")
	return nil
}

func disable() {
	if saved != nil {
		_, _ = os.Stdout.WriteString("\x1b[?25h\x1b[?1049l")
		_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(int(os.Stdin.Fd())), uintptr(tcssets),
			uintptr(unsafe.Pointer(saved)))
		saved = nil
	}
}

func termSize() (int, int) {
	var ws winsz
	if err := ioctl(winsize, int(os.Stdout.Fd()), unsafe.Pointer(&ws)); err != 0 || ws.Col == 0 {
		return 80, 24
	}
	return int(ws.Col), int(ws.Row)
}

// nextKey blocks until one actionable key arrives. Sequences are parsed off a
// carry buffer, because a terminal frequently delivers a multi-byte escape and
// the next keystroke in the same read.
func nextKey() (Key, error) {
	fd := int(os.Stdin.Fd())
	for {
		// A decisive scan that yields no key still falls through to the read: an
		// empty carry buffer is a verdict (ok=true), so looping here would spin
		// and never read stdin at all, leaving a picker that answers to nothing.
		if k, rest, ok := scan(keyBuf); ok {
			keyBuf = rest
			if k != KeyNone {
				return k, nil
			}
		}
		var b [256]byte
		// syscall.Read, not os.Stdin.Read: os.File turns a zero-byte read into
		// io.EOF, and with VMIN=0/VTIME=1 a read timeout IS a zero-byte read.
		// Wrapped, nextKey reported a fatal error a hundred milliseconds after the
		// last keypress and the key reader died - the picker then sat there deaf,
		// and with no further keypresses at all it deadlocked outright. The raw
		// read leaves a timeout as a nil error, so it stays distinct from a
		// terminal that is actually gone. fd was taken once at the top, because
		// os.File.Fd() fcntls the descriptor on every call.
		n, err := syscall.Read(fd, b[:])
		if err == syscall.EINTR {
			// A resize interrupts the read; the key is still queued.
			continue
		}
		if err != nil {
			return KeyNone, err
		}
		if n == 0 {
			// The short read timed out, so whatever is buffered is complete:
			// a lone ESC is an Escape now rather than the first byte of a
			// sequence that never arrived.
			if k, rest, ok := scan(keyBuf); ok && k != KeyNone {
				keyBuf = rest
				return k, nil
			}
			continue
		}
		keyBuf = append(keyBuf, b[:n]...)
	}
}
