//go:build darwin || freebsd || dragonfly || netbsd || openbsd

package tui

import "syscall"

// BSD encodes an ioctl as direction | length<<16 | group<<8 | number. These are
// looked up, not composed. The request number for termios is 0x13, and this
// used to hardcode 0x68 - which is TIOCGWINSZ's number. The old tcgets came out
// as 0x40487468 on darwin, a request no kernel registers, so the call failed
// with ENOTTY and isTerminal() read that as "not a terminal": `lxmusic search`
// printed its table and exited instead of opening the picker.
//
// syscall generates these from the platform headers, so the number and the
// length it encodes stay in step with syscall.Termios: darwin's termios is 72
// bytes (0x48) and every other BSD's is 44 (0x2c), and the constants already
// carry the right figure for each. linux keeps its own file, because stdlib
// does not export TCGETS there.
const (
	tcgets  = syscall.TIOCGETA
	tcssets = syscall.TIOCSETA
	winsize = syscall.TIOCGWINSZ
)
