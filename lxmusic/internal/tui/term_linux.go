//go:build linux

package tui

// Linux encodes ioctl requests differently from the BSDs: a single byte
// holding the type and direction bits plus a three-byte number. stdlib exposes
// no termios API here, so the two are looked up by their published values.
const (
	tcgets  = 0x5401 // TCGETS
	tcssets = 0x5402 // TCSETS
)
