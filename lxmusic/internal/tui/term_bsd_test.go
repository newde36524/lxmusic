//go:build darwin || freebsd || dragonfly || netbsd || openbsd

package tui

import (
	"syscall"
	"testing"
	"unsafe"
)

// BSD packs an ioctl as direction | length<<16 | group<<8 | number, where the
// length field is 14 bits, not 8 - the low eight bits are the request number,
// so 0x3fff0000 is the mask, and 0x003f0000 silently reads 8 instead of 72.
//
// Two invariants, both of which broke when these were composed by hand:
//
// The length field must equal the struct the kernel checks against, or the
// request is meaningless. And the number must be the termios number, not
// winsize's - before, darwin's tcgets read 0x40487468, which is TIOCGWINSZ
// carrying a 72-byte length. No kernel registers that, the call failed with
// ENOTTY, isTerminal() read it as "not a terminal", and `lxmusic search`
// printed its table and exited instead of opening the picker.
func TestBsdIoctlEncoding(t *testing.T) {
	const (
		lenMask = uint32(0x3fff0000)
		numMask = uint32(0x000000ff)
	)
	if got, want := (uint32(tcgets)&lenMask)>>16, uint32(unsafe.Sizeof(syscall.Termios{})); got != want {
		t.Errorf("tcgets encodes a termios of %d bytes, syscall.Termios is %d", got, want)
	}
	if got, want := (uint32(winsize)&lenMask)>>16, uint32(unsafe.Sizeof(winsz{})); got != want {
		t.Errorf("winsize encodes a winsize of %d bytes, winsz is %d", got, want)
	}
	if got := uint32(tcgets) & numMask; got == uint32(winsize)&numMask {
		t.Errorf("tcgets and winsize share request number 0x%x: tcgets is not the termios request", got)
	}
}
