// Command spawn isolates which CreateProcessW argument breaks on this machine.
//
// It is a probe, not product code.
package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	createNewConsole = 0x00000010
	createBreakaway  = 0x01000000
	noWindow         = 0x08000000

	startfUseShowWindow = 0x00000001
	swHide              = 0
	swShow              = 5
)

type startupInfo struct {
	cb                uint32
	_pad              uint32
	lpReserved        unsafe.Pointer
	lpTitle           unsafe.Pointer
	dwX               uint32
	dwY               uint32
	dwWindowX         uint32
	dwWindowY         uint32
	dwWindowWidth     uint32
	dwWindowHeight    uint32
	dwWindowPlacement uint32
	dwFlags           uint32
	dwShowWindow      uint32
	wReserved         [2]uint16
	lpReserved2       unsafe.Pointer
	hStdInput         unsafe.Pointer
	hStdOutput        unsafe.Pointer
	hStdError         unsafe.Pointer
}

var (
	k32            = syscall.NewLazyDLL("kernel32.dll")
	pCreateProcess = k32.NewProc("CreateProcessW")
	pWaitFor       = k32.NewProc("WaitForSingleObject")
	pGetExitCode   = k32.NewProc("GetExitCodeProcess")
	pCloseHandle   = k32.NewProc("CloseHandle")
)

// try spawns cmd once and reports what happened. Empty app or cwd means NULL.
func try(name, cmd, app, cwd string, flags uintptr, dwFlags, dwShow uint32) {
	lp, _ := syscall.UTF16PtrFromString(cmd)
	var pApp, pCwd unsafe.Pointer
	if app != "" {
		if s, _ := syscall.UTF16PtrFromString(app); s != nil {
			pApp = unsafe.Pointer(s)
		}
	}
	if cwd != "" {
		if s, _ := syscall.UTF16PtrFromString(cwd); s != nil {
			pCwd = unsafe.Pointer(s)
		}
	}
	var si startupInfo
	si.cb = uint32(unsafe.Sizeof(si))
	si.dwFlags = dwFlags
	si.dwShowWindow = dwShow
	var pi [4]uintptr
	r, _, e := pCreateProcess.Call(uintptr(pApp), uintptr(unsafe.Pointer(lp)), 0, 0, 0,
		flags, 0, uintptr(pCwd), uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)))
	if r == 0 {
		fmt.Printf("%-40s CreateProcessW FAILED: %v\n", name, e)
		return
	}
	w, _, _ := pWaitFor.Call(pi[0], 15000)
	var code uint32
	pGetExitCode.Call(pi[0], uintptr(unsafe.Pointer(&code)))
	pCloseHandle.Call(pi[0])
	pCloseHandle.Call(pi[1])
	state := "still alive"
	if w != 0 {
		state = fmt.Sprintf("exited code=%d", code)
	}
	fmt.Printf("%-40s OK pid=%d %s\n", name, uint32(pi[2]), state)
}

func main() {
	cmd := `C:\Windows\System32\cmd.exe /c exit 0`
	const app = `C:\Windows\System32\cmd.exe`

	fmt.Printf("STARTUPINFO size = %d (want 96)\n", unsafe.Sizeof(startupInfo{}))
	fmt.Println()

	try("flags=0", cmd, "", "", 0, 0, 0)
	try("flags=0 + app", cmd, app, "", 0, 0, 0)
	try("flags=0 + cwd System32", cmd, "", `C:\Windows\System32`, 0, 0, 0)
	try("flags=0 + cwd C:\\lx-rl", cmd, "", `C:\lx-rl`, 0, 0, 0)
	try("flags=0 + app + cwd", cmd, app, `C:\lx-rl`, 0, 0, 0)
	fmt.Println()
	try("CREATE_NEW_CONSOLE", cmd, "", "", uintptr(createNewConsole), 0, 0)
	try("NEW_CONSOLE + app + cwd", cmd, app, `C:\lx-rl`, uintptr(createNewConsole), 0, 0)
	try("NEW_CONSOLE + HIDE", cmd, "", "", uintptr(createNewConsole), startfUseShowWindow, swHide)
	try("NEW_CONSOLE + SHOW", cmd, "", "", uintptr(createNewConsole), startfUseShowWindow, swShow)
	try("NEW_CONSOLE + NO_WINDOW", cmd, "", "", uintptr(createNewConsole|noWindow), 0, 0)
	fmt.Println()
	try("BREAKAWAY_FROM_JOB", cmd, "", "", uintptr(createBreakaway), 0, 0)
	try("NEW_CONSOLE + BREAKAWAY", cmd, "", "", uintptr(createNewConsole|createBreakaway), 0, 0)
	try("NO_WINDOW", cmd, "", "", uintptr(noWindow), 0, 0)

	const lx = `C:\lx-rl\bin\lxmusic.exe`
	fmt.Println()
	fmt.Printf("lxmusic.exe exists = %v\n", exists(lx))
	try("lxmusic version", lx+` version`, "", "", 0, 0, 0)
	try("lxmusic version + NEW_CONSOLE", lx+` version`, "", "",
		uintptr(createNewConsole), 0, 0)
	try("lxmusic, name+line separated", lx+` version`, lx, `C:\lx-rl`,
		uintptr(createNewConsole), 0, 0)
	try("lxmusic, line only", lx+` version`, "", "",
		uintptr(createNewConsole), 0, 0)
}

func exists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
