//go:build linux || darwin || freebsd || dragonfly || netbsd || openbsd

package tui

import (
	"os"
	"syscall"
	"unsafe"
)

// winsz is struct winsize: four uint16, on every unix.
type winsz struct{ Row, Col, Xpixel, Ypixel uint16 }

const (
	// The three lflag bits this picker needs have kept their 4.2BSD values on
	// every unix that ships termios(4) since: they never moved. OPOST and IXON
	// do move between platforms, so they are left alone — nothing in this
	// renderer depends on either, because it never writes a newline.
	fEcho  = 0x8
	fCanon = 0x2
	fSig   = 0x1
)

// The per-platform values (ioctl numbers, VMIN/VTIME slots) come from the
// term_* files. syscall.Syscall and syscall.SYS_IOCTL are the only termios
// entry points the standard library keeps identical on every unix, so this is
// what makes the layer portable with no cgo and no third-party module.
func ioctl(trap uintptr, fd int, p unsafe.Pointer) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), trap, uintptr(p))
	return err
}

var saved *syscall.Termios

// isTerminal is true only when both standard streams are real terminals.
// Redirected or piped output must never receive escape sequences.
func isTerminal() bool {
	var t syscall.Termios
	if err := ioctl(tcgets, int(os.Stdout.Fd()), unsafe.Pointer(&t)); err != nil {
		return false
	}
	return int(os.Stdin.Fd()) >= 0
}

// enable drops the terminal into raw mode so keystrokes arrive immediately.
// Only ECHO, ICANON and ISIG are cleared: they hold their 4.2BSD values on
// every unix, and they are the three that matter here. OPOST and IXON move
// between platforms, and nothing in this renderer needs them changed.
//
// VMIN is 0 and VTIME is 1 (100 ms): a read returns at once when a byte is
// there, but times out instead of blocking forever. That timeout is what lets
// nextKey tell a lone ESC from the start of a sequence that was split across
// reads, without adding latency to every ordinary keypress.
func enable() error {
	fd := int(os.Stdin.Fd())
	var old syscall.Termios
	if err := ioctl(tcgets, fd, unsafe.Pointer(&old)); err != nil {
		return err
	}
	raw := old
	raw.Lflag &^= fEcho | fCanon | fSig
	raw.Cc[ccVmin] = 0
	raw.Cc[ccVtime] = 1
	if err := ioctl(tcssets, fd, unsafe.Pointer(&raw)); err != nil {
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
	if err := ioctl(winsize, int(os.Stdout.Fd()), unsafe.Pointer(&ws)); err != nil || ws.Col == 0 {
		return 80, 24
	}
	return int(ws.Col), int(ws.Row)
}

// nextKey blocks until one actionable key arrives. Sequences are parsed off a
// carry buffer, because a terminal frequently delivers a multi-byte escape and
// the next keystroke in the same read.
func nextKey() (Key, error) {
	for {
		if k, rest, ok := scan(keyBuf); ok {
			keyBuf = rest
			if k != KeyNone {
				return k, nil
			}
			continue
		}
		var b [256]byte
		n, err := os.Stdin.Read(b[:])
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
