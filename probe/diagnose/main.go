// Command diagnose reports exactly what this session permits: which console API
// sets the loader can resolve, whether a console can be created at all, and
// what the standard handles point at. Written because the console probe failed
// with no reason given, and the reason determines how much of the picker can be
// verified here.
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func main() {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	lp := k32.NewProc("LoadLibraryW")
	lpEx := k32.NewProc("LoadLibraryExW")
	pAlloc := k32.NewProc("AllocConsole")
	pFree := k32.NewProc("FreeConsole")
	pGetLastErr := k32.NewProc("GetLastError")
	pGetStd := k32.NewProc("GetStdHandle")
	pGetMode := k32.NewProc("GetConsoleMode")
	pFreeLib := k32.NewProc("FreeLibrary")

	const sys32 = 0x00000800 // LOAD_LIBRARY_SEARCH_SYSTEM32
	const (
		stdIn  = ^uintptr(0) - 9  // STD_INPUT_HANDLE  == -10
		stdOut = ^uintptr(0) - 10 // STD_OUTPUT_HANDLE == -11
		stdErr = ^uintptr(0) - 8  // STD_ERROR_HANDLE  == -12
	)

	lastErr := func() uint32 {
		e, _, _ := pGetLastErr.Call()
		return uint32(e)
	}

	fmt.Println("== LoadLibrary on console API sets ==")
	for _, n := range []string{
		`api-ms-win-console-pty-l1-1-0.dll`,
		`api-ms-win-core-console-l1-2-0.dll`,
		`api-ms-win-core-heap-l1-1-0.dll`,
		`kernel32.dll`,
	} {
		w, _ := syscall.UTF16PtrFromString(n)
		h, _, _ := lp.Call(uintptr(unsafe.Pointer(w)))
		err := lastErr()
		fmt.Printf("  LoadLibraryW  %-40s h=%-5v lastErr=%d\n", n, h != 0, err)
	}

	fmt.Println("== restricted vs unrestricted search for the PTY API set ==")
	w, _ := syscall.UTF16PtrFromString(`api-ms-win-console-pty-l1-1-0.dll`)
	h1, _, _ := lpEx.Call(uintptr(unsafe.Pointer(w)), 0, sys32)
	fmt.Printf("  LoadLibraryExW SEARCH_SYSTEM32 = %v\n", h1 != 0)
	if h1 != 0 {
		pFreeLib.Call(h1)
	}
	h2, _, _ := lpEx.Call(uintptr(unsafe.Pointer(w)), 0, 0)
	fmt.Printf("  LoadLibraryExW flags=0         = %v\n", h2 != 0)
	if h2 != 0 {
		pFreeLib.Call(h2)
	}

	fmt.Println("== can this process get a console at all ==")
	r, _, _ := pAlloc.Call()
	err := lastErr()
	fmt.Printf("  AllocConsole -> %v, GetLastError=%d\n", r != 0, err)
	if r != 0 {
		pFree.Call()
	}

	fmt.Println("== standard handles ==")
	for _, hh := range []struct {
		name string
		n    uintptr
	}{
		{"STD_INPUT_HANDLE", stdIn},
		{"STD_OUTPUT_HANDLE", stdOut},
		{"STD_ERROR_HANDLE", stdErr},
	} {
		h, _, _ := pGetStd.Call(hh.n)
		var mode uint32
		m, _, _ := pGetMode.Call(h, uintptr(unsafe.Pointer(&mode)))
		fmt.Printf("  %-20s handle=%-5v GetConsoleMode=%-5v mode=0x%X\n",
			hh.name, h != 0, m != 0, mode)
	}
}
