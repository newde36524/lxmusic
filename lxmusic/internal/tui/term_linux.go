//go:build linux

package tui

const (
	// Linux encodes ioctl requests differently from the BSDs: a single byte
	// holding the type and direction bits plus a three-byte number.
	tcgets  = 0x5401
	tcssets = 0x5402
	winsize = 0x5413

	// Linux follows the SVR4 slot order for the Cc table.
	ccVmin  = 6
	ccVtime = 5
)
