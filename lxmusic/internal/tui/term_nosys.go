//go:build !windows && !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd

package tui

import "os"

// Go's syscall package exposes no termios API on this GOOS, so the interactive
// picker cannot be driven. Reporting no terminal makes the caller fall back to
// the plain table rather than emit escape sequences into a non-tty.
func isTerminal() bool { return false }

func enable() error { return os.ErrInvalid }

func disable() {}

func termSize() (int, int) { return 80, 24 }

func nextKey() (Key, error) { return KeyNone, os.ErrInvalid }
