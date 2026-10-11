//go:build freebsd || dragonfly || openbsd

package tui

// The original 4.2BSD kernels swapped these two Cc slots relative to SVR4.
const (
	ccVmin  = 5
	ccVtime = 6
)
