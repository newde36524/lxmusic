//go:build darwin || freebsd || dragonfly || netbsd || openbsd

package tui

import (
	"syscall"
	"unsafe"
)

const (
	tioGroup = 0x74 // 't', the termios ioctl group
	tioNr    = 0x68 // TIOCGETA / TIOCSETA request number
	tioLen   = unsafe.Sizeof(syscall.Termios{})
	wszLen   = unsafe.Sizeof(winsz{})

	// BSD encodes an ioctl as direction | length<<16 | group<<8 | number. The
	// lengths are taken from the real structs, so they cannot drift from the
	// ABI the kernel checks against. TIOCGETA is a read, which sets bit 30;
	// TIOCSETA is a write, which sets bit 31.
	tcgets  = 0x40000000 | (tioLen << 16) | (tioGroup << 8) | tioNr
	tcssets = 0x80000000 | (tioLen << 16) | (tioGroup << 8) | tioNr
	winsize = 0x40000000 | (wszLen << 16) | (tioGroup << 8) | tioNr
)
