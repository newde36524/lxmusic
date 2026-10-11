//go:build darwin || netbsd

package tui

// These kernels follow the SVR4 order for the two Cc slots this picker sets.
const (
	ccVmin  = 6
	ccVtime = 5
)
