//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	cpUTF8 = 65001

	// The standard handles are negative decimal sentinels, written in
	// two's-complement form rather than as a negative constant for the same
	// reason as in term_windows.go.
	stdOut  = ^uintptr(0) - 10
	invalid = ^uintptr(0)
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	pGetStdHandle       = kernel32.NewProc("GetStdHandle")
	pGetConsoleMode     = kernel32.NewProc("GetConsoleMode")
	pGetConsoleOutputCP = kernel32.NewProc("GetConsoleOutputCP")
	pSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
)

// setupConsole makes the console render the program's UTF-8 output. Go writes
// UTF-8 bytes straight into the console handle, and the console decodes them
// with its output codepage - on a Chinese system that is 936 by default, so
// this program's own Chinese labels would print as mojibake on Windows while
// coming out clean on a mac, whose terminal is UTF-8 by construction. This is
// the one place the two platforms would otherwise disagree on the same bytes.
//
// The input codepage is deliberately left alone: the interactive keys arrive
// through ReadConsoleInputW as Unicode and never pass through it.
//
// The change is confined to this process's run. A console outlives the process
// that changed it, so the previous codepage is captured and restored by the
// returned function. Nothing is touched when stdout is not a real console -
// the bytes would go to a pipe and the console's codepage would be the shell's
// business - or when it is already at 65001, or when it refuses the change,
// which is still the case on some older builds. The returned function is
// always non-nil - a no-op when nothing was changed - so the caller can call
// it without branching.
func setupConsole() func() {
	// The nothing-changed value is a real function, never nil: main calls the
	// result immediately before os.Exit, and os.Exit skips defers, so there is
	// no other place to do it. A nil closure would panic there.
	restore := func() {}
	h, _, _ := pGetStdHandle.Call(stdOut)
	if h == invalid {
		return restore
	}
	var mode uint32
	if r, _, _ := pGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode))); r == 0 {
		// A pipe or a redirected file: the console's codepage is not ours to set.
		return restore
	}
	old, _, _ := pGetConsoleOutputCP.Call()
	if old == cpUTF8 {
		return restore
	}
	if r, _, _ := pSetConsoleOutputCP.Call(cpUTF8); r == 0 {
		return restore
	}
	return func() { pSetConsoleOutputCP.Call(old) }
}
