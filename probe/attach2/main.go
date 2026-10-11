// Command attach2 asks one question: can a descendant reach a console that a
// sibling created, and through which route?
//
// AllocConsole and AttachConsole both fail here with ERROR_ACCESS_DENIED, which
// is the process-group or job restriction. A process in a protected process
// group (conhost, cmd) is exempt from group membership, so a hop through
// cmd.exe might put the attaching process outside the group.
//
// It is a probe, not product code.
//
// Modes:
//
//	attach2                 spawn a sibling with CREATE_NEW_CONSOLE, then try
//	                         to reach its console directly and via a cmd hop
//	attach2 --attach <pid>  attach, open CONIN$/CONOUT$, report each result
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

const (
	createNewConsole = 0x00000010
	genericRead      = uintptr(0x80000000)
	genericWrite     = uintptr(0x40000000)
	shareAll         = uintptr(0x00000003)
	openExisting     = uintptr(3)
	stillActive      = 259
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
	k32 = syscall.NewLazyDLL("kernel32.dll")

	pCreateProcess  = k32.NewProc("CreateProcessW")
	pAttachConsole  = k32.NewProc("AttachConsole")
	pFreeConsole    = k32.NewProc("FreeConsole")
	pCreateFileW    = k32.NewProc("CreateFileW")
	pGetConsoleMode = k32.NewProc("GetConsoleMode")
	pGetExitCode    = k32.NewProc("GetExitCodeProcess")
	pTerminate      = k32.NewProc("TerminateProcess")
	pCloseHandle    = k32.NewProc("CloseHandle")
)

func u16(s string) uintptr {
	p, _ := syscall.UTF16PtrFromString(s)
	return uintptr(unsafe.Pointer(p))
}

func spawn(cmdLine string, flags uintptr) (uint32, uintptr, error) {
	var si startupInfo
	si.cb = uint32(unsafe.Sizeof(si))
	var pi [4]uintptr
	r, _, e := pCreateProcess.Call(0, u16(cmdLine), 0, 0, 1, flags, 0, 0,
		uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	if r == 0 {
		return 0, 0, e
	}
	return uint32(pi[2]), pi[0], nil
}

// attachOnly reports whether this process can attach to pid's console, and if
// so whether CONIN$ and CONOUT$ come back as usable console handles.
func attachOnly(pid uint32) {
	tag := fmt.Sprintf("attach2 --attach %d", pid)
	r, _, e := pAttachConsole.Call(uintptr(pid))
	if r == 0 {
		fmt.Printf("%s -> AttachConsole FAILED: %v\n", tag, e)
		return
	}
	fmt.Printf("%s -> AttachConsole OK\n", tag)
	defer pFreeConsole.Call()

	for _, tc := range []struct {
		name string
		acc  uintptr
	}{
		{"CONOUT$", genericRead},
		{"CONIN$", genericRead | genericWrite},
	} {
		h, _, e := pCreateFileW.Call(u16(tc.name), tc.acc, shareAll, 0,
			openExisting, 0, 0)
		if h == 0 || h == ^uintptr(0) {
			fmt.Printf("%s   CreateFileW(%-7s) FAILED: %v\n", tag, tc.name, e)
			continue
		}
		var mode uint32
		if r, _, e := pGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode))); r == 0 {
			fmt.Printf("%s   %-8s handle 0x%X, NOT a console: %v\n",
				tag, tc.name, h, e)
		} else {
			fmt.Printf("%s   %-8s handle 0x%X, console mode 0x%X\n",
				tag, tc.name, h, mode)
		}
		pCloseHandle.Call(h)
	}
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--attach" {
		pid, _ := strconv.ParseUint(os.Args[2], 10, 32)
		attachOnly(uint32(pid))
		return
	}

	sib, hSib, err := spawn(
		`C:\Windows\System32\cmd.exe /c ping -n 600 127.0.0.1`,
		uintptr(createNewConsole))
	if err != nil {
		fmt.Printf("sibling spawn FAILED: %v\n", err)
		os.Exit(1)
	}
	time.Sleep(1500 * time.Millisecond)
	var code uint32
	pGetExitCode.Call(hSib, uintptr(unsafe.Pointer(&code)))
	fmt.Printf("sibling pid=%d exitcode=%d (259 = still running)\n\n", sib, code)
	defer func() {
		pTerminate.Call(hSib, 0)
		pCloseHandle.Call(hSib)
	}()

	fmt.Println("-- R1: attach directly from this process --")
	attachOnly(sib)
	pFreeConsole.Call()
	fmt.Println()

	self, err := exec.LookPath(os.Args[0])
	if err != nil {
		fmt.Printf("cannot locate myself: %v\n", err)
		return
	}
	// R2: hop through cmd.exe. cmd.exe is in a protected process group and is
	// exempt from the caller's group, so its child may not be in ours.
	fmt.Printf("-- R2: attach from a child of cmd.exe (protected group hop) --\n")
	c := exec.Command("cmd.exe", "/c",
		fmt.Sprintf("%s --attach %d", self, sib))
	if out, e := c.CombinedOutput(); e != nil {
		fmt.Printf("%s [exit %v]\n", string(out), e)
	} else {
		fmt.Println(string(out))
	}
}
