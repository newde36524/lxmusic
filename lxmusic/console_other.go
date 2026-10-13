//go:build !windows

package main

// setupConsole is a no-op here. unix terminals are byte-transparent: what Go
// writes is what the terminal receives, and they are UTF-8 by convention, so
// there is nothing to negotiate. The call still exists so main does not need to
// know which side of the platforms it is on.
func setupConsole() func() { return func() {} }
