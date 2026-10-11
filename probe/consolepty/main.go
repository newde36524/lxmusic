// Command consolepty drives lxmusic.exe inside a real Windows console and reads
// back what it rendered.
//
// There is no ConPTY on this box: api-ms-win-conpty-l1-1-0.dll is not present
// in System32, `wsl -l` has no distro, and there is no winpty. AllocConsole also
// refuses here with ERROR_ACCESS_DENIED, and the process-group workaround is
// unavailable too - this kernel32 does not export GetCurrentProcessThreadAttributeValue
// and UpdateProcThreadAttribute rejects both PROCESS_GROUP_AFFINITY and
// PARENT_PROCESS with ERROR_NOT_SUPPORTED. So the parent cannot create or leave
// a group it is in.
//
// The route that needs neither is CREATE_NEW_CONSOLE: the child gets its own
// console, the parent then AttachConsole(pid)s onto that same one and reaches
// it through CONIN$ and CONOUT$. The child has real console handles on stdin and
// stdout, so isatty() is true and the picker comes up instead of falling back to
// the plain table.
//
// Input is injected with WriteConsoleInputW using the exact INPUT_RECORD layout
// term_windows.go decodes (type@0, bKeyDown@4, VK@10, char@14, state@16), and
// output is read with ReadConsoleOutputW - the rendered grid and its character
// attributes, which is what a person would actually see. bKeyDown is set
// honestly on each half of a press, so if the child still moves one row per tap
// the timing filter is doing the work, and no field at offset 4 or 8 is.
//
// It is a probe, not product code. Nothing in the project imports it.
//
// Usage: consolepty <lxmusic.exe> <outDir> <logPath>
package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	stdIn  = ^uintptr(0) - 9  // STD_INPUT_HANDLE == -10
	stdOut = ^uintptr(0) - 10 // STD_OUTPUT_HANDLE == -11

	createNewConsole = 0x00000010

	genericRead  = uintptr(0x80000000)
	genericWrite = uintptr(0x40000000)
	shareAll     = uintptr(0x00000003)
	openExisting = uintptr(3)

	startfUseShowWindow = 0x00000001
	swHide              = 0

	keyEvent = 1

	vkUp     = 0x26
	vkDown   = 0x28
	vkPrior  = 0x21
	vkNext   = 0x22
	vkHome   = 0x24
	vkEnd    = 0x23
	vkReturn = 0x0d
	vkEscape = 0x1b

	enhanced = 0x0100

	offEventType = 0
	offKeyDown   = 4
	offRepeat    = 8
	offVK        = 10
	offScan      = 12
	offUChar     = 14
	offKeyState  = 16
	recLen       = 20

	constCols = 110
	constRows = 30
)

var (
	k32 = syscall.NewLazyDLL("kernel32.dll")
	u32 = syscall.NewLazyDLL("user32.dll")

	pGetStdHandle   = k32.NewProc("GetStdHandle")
	pGetConsoleMode = k32.NewProc("GetConsoleMode")
	pReadConsoleOut = k32.NewProc("ReadConsoleOutputW")
	pWriteConsoleIn = k32.NewProc("WriteConsoleInputW")
	pSetScreenSize  = k32.NewProc("SetConsoleScreenBufferSize")
	pSetWindowInfo  = k32.NewProc("SetConsoleWindowInfo")
	pAttachConsole  = k32.NewProc("AttachConsole")
	pFreeConsole    = k32.NewProc("FreeConsole")
	pCreateFileW    = k32.NewProc("CreateFileW")
	pCreateProcess  = k32.NewProc("CreateProcessW")
	pWaitFor        = k32.NewProc("WaitForSingleObject")
	pGetExitCode    = k32.NewProc("GetExitCodeProcess")
	pCloseHandle    = k32.NewProc("CloseHandle")
	pGetConsoleWin  = u32.NewProc("GetConsoleWindow")
)

// STARTUPINFO, laid out as winbase.h lays it out. dwFlags sits at 48 and
// dwShowWindow at 52; with lpReserved and lpTitle both present the total is 96,
// and STARTUPINFOEX is 96 + 8 + 8 = 112.
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

func packCoord(x, y int16) uintptr {
	return uintptr(uint16(x)) | uintptr(uint16(y))<<16
}

func le16(b []byte, off int) uint16 {
	return uint16(b[off]) | uint16(b[off+1])<<8
}

type cell struct {
	ch   uint16
	attr uint16
}

// sess is the parent's view of the console the child is drawing into.
type sess struct {
	hIn  uintptr
	hOut uintptr
	cols int
	rows int
	log  *os.File
}

func (s *sess) say(format string, a ...any) {
	fmt.Fprintf(s.log, format+"\n", a...)
}

// screen returns the rendered grid: rows of cells with their console attributes.
func (s *sess) screen() [][]cell {
	n := s.cols * s.rows
	buf := make([]byte, n*4)
	var reg [4]int16
	reg[2], reg[3] = int16(s.cols-1), int16(s.rows-1)

	r, _, _ := pReadConsoleOut.Call(s.hOut, uintptr(unsafe.Pointer(&buf[0])),
		packCoord(int16(s.cols), int16(s.rows)), packCoord(0, 0),
		uintptr(unsafe.Pointer(&reg[0])))
	if r == 0 {
		return nil
	}
	grid := make([][]cell, s.rows)
	for y := 0; y < s.rows; y++ {
		row := make([]cell, s.cols)
		for x := 0; x < s.cols; x++ {
			i := (y*s.cols + x) * 4
			row[x] = cell{ch: le16(buf, i), attr: le16(buf, i+2)}
		}
		grid[y] = row
	}
	return grid
}

// text renders the grid as plain lines, trailing blanks trimmed.
func (s *sess) text() string {
	var b strings.Builder
	for _, row := range s.screen() {
		var line strings.Builder
		for _, c := range row {
			if c.ch == ' ' || c.ch == 0 {
				line.WriteByte(' ')
			} else {
				line.WriteRune(rune(c.ch))
			}
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// highlightRows returns the screen rows painted in reverse video, i.e. the rows
// the picker selected. Conhost turns \x1b[7m into a swapped attribute with the
// background-intensity bit set, so any attribute other than the plain 7 marks a
// reversed cell. A row counts when most of it is reversed, which keeps the
// prompt line's short reversed marker out.
func highlightRows(g [][]cell) []int {
	var out []int
	for y, row := range g {
		hit := 0
		for _, c := range row {
			if c.ch != ' ' && c.ch != 0 && c.attr != 0 && c.attr != 7 {
				hit++
			}
		}
		if hit*2 > len(row) {
			out = append(out, y)
		}
	}
	return out
}

func (s *sess) dump(what string) {
	s.say("--- screen after %s ---", what)
	for _, line := range strings.Split(s.text(), "\n") {
		s.say("   %s", line)
	}
}

// rawKey writes one INPUT_RECORD half into the console input queue.
func (s *sess) rawKey(vk, ch uint16, state uint32, down bool) {
	b := make([]byte, recLen)
	b[offEventType] = byte(keyEvent)
	b[offVK] = byte(vk)
	b[offVK+1] = byte(vk >> 8)
	b[offUChar] = byte(ch)
	b[offUChar+1] = byte(ch >> 8)
	if down {
		b[offKeyDown] = 1
	}
	b[offRepeat] = 1
	b[offScan] = byte(vk)
	b[offKeyState] = byte(state)
	b[offKeyState+1] = byte(state >> 8)
	b[offKeyState+2] = byte(state >> 16)
	var n uint32
	pWriteConsoleIn.Call(s.hIn, 1, uintptr(unsafe.Pointer(&b[0])),
		uintptr(unsafe.Pointer(&n)))
}

// press injects one physical key press: a key-down half, then, a couple of
// hundred microseconds later, the key-up half the console always follows it
// with. Both halves carry the same VK and character.
func (s *sess) press(vk, ch uint16, state uint32) {
	s.rawKey(vk, ch, state, true)
	time.Sleep(200 * time.Microsecond)
	s.rawKey(vk, ch, state, false)
}

func (s *sess) wait(what string, d time.Duration, fn func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(120 * time.Millisecond)
	}
	s.say("TIMEOUT: %s", what)
	return false
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: consolepty <lxmusic.exe> <outDir> <logPath>")
		os.Exit(2)
	}
	binary, outDir, logPath := os.Args[1], os.Args[2], os.Args[3]

	log, err := os.Create(logPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "log:", err)
		os.Exit(2)
	}
	defer log.Close()
	fail := false
	say := func(format string, a ...any) {
		fmt.Fprintf(log, format+"\n", a...)
	}
	kill := func(format string, a ...any) {
		say("FAIL: "+format, a...)
		fail = true
	}
	say("binary : %s", binary)
	say("outDir : %s", outDir)
	say("build  : %s", time.Now().Format(time.RFC3339))

	// ---- phase 0: the child gets its own console, and we attach to it -----
	cmdLine := binary + ` search 晴天 --source kw --limit 8 --out ` + outDir
	lp, err := syscall.UTF16PtrFromString(cmdLine)
	if err != nil {
		say("FAIL: utf16: %v", err)
		os.Exit(1)
	}
	app, err := syscall.UTF16PtrFromString(binary)
	if err != nil {
		say("FAIL: utf16 app: %v", err)
		os.Exit(1)
	}
	cwd, err := syscall.UTF16PtrFromString(`C:\lx-rl`)
	if err != nil {
		say("FAIL: utf16 cwd: %v", err)
		os.Exit(1)
	}

	var si startupInfo
	si.cb = uint32(unsafe.Sizeof(si))
	si.dwFlags = startfUseShowWindow
	si.dwShowWindow = swHide
	var pi [4]uintptr // hProcess, hThread, dwProcessId, dwThreadId
	var r uintptr
	r, _, err = pCreateProcess.Call(uintptr(unsafe.Pointer(app)),
		uintptr(unsafe.Pointer(lp)), 0, 0, 0, uintptr(createNewConsole), 0,
		uintptr(unsafe.Pointer(cwd)),
		uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	if r == 0 {
		say("FAIL: CreateProcessW(CREATE_NEW_CONSOLE): %v", err)
		os.Exit(1)
	}
	pid := uint32(pi[2]) // PROCESS_INFORMATION.dwProcessId
	say("phase 0: child started, pid=%d", pid)
	say("phase 0 cmdline: %s", cmdLine)
	if err := pGetConsoleWin.Find(); err != nil {
		say("phase 0 note: user32.GetConsoleWindow unavailable (%v)", err)
	} else {
		cw, _, _ := pGetConsoleWin.Call()
		say("phase 0: child console window=0x%X", cw)
	}

	// CREATE_NEW_CONSOLE creates the console inside the child; if that fails the
	// child dies at once, so catch it here instead of stalling on the render.
	wc, _, _ := pWaitFor.Call(pi[0], 5000)
	if wc == 0 {
		say("FAIL: the child exited within 5 seconds of starting")
		var code uint32
		pGetExitCode.Call(pi[0], uintptr(unsafe.Pointer(&code)))
		say("FAIL: child exit code = %d", code)
		pCloseHandle.Call(pi[0])
		pCloseHandle.Call(pi[1])
		os.Exit(1)
	}

	// The child needs a moment to attach its console before we can.
	attached := false
	var attachErr error
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if r, _, e := pAttachConsole.Call(uintptr(pid)); r != 0 {
			attached = true
			break
		} else if attachErr == nil {
			attachErr = e
		}
		time.Sleep(150 * time.Millisecond)
	}
	// Child state at this moment tells us whether the console even exists.
	wc, _, _ = pWaitFor.Call(pi[0], 0)
	var cexit uint32
	pGetExitCode.Call(pi[0], uintptr(unsafe.Pointer(&cexit)))
	say("  child at attach time: alive=%v exitcode=%d", wc == 0 || wc == 259, cexit)
	if !attached {
		say("FAIL: AttachConsole(%d) never succeeded: %v", pid, attachErr)
		pCloseHandle.Call(pi[0])
		pCloseHandle.Call(pi[1])
		os.Exit(1)
	}
	defer pFreeConsole.Call()

	// After AttachConsole our standard handles are still the pipes we were
	// launched with, so the console has to be opened by name.
	var hIn, hOut uintptr
	for _, tc := range []struct {
		name *string
		acc  uintptr
		out  *uintptr
	}{
		{stringPtr("CONOUT$"), genericRead, &hOut},
		{stringPtr("CONIN$"), genericRead | genericWrite, &hIn},
	} {
		lpName, _ := syscall.UTF16PtrFromString(*tc.name)
		h, _, e := pCreateFileW.Call(uintptr(unsafe.Pointer(lpName)), tc.acc,
			shareAll, 0, openExisting, 0, 0)
		if h == 0 {
			say("FAIL: CreateFileW(%s): %v", *tc.name, e)
			os.Exit(1)
		}
		*tc.out = h
	}
	var mode uint32
	if r, _, _ = pGetConsoleMode.Call(hOut, uintptr(unsafe.Pointer(&mode))); r == 0 {
		say("FAIL: CONOUT$ is not a console handle")
		os.Exit(1)
	}
	if r, _, _ = pGetConsoleMode.Call(hIn, uintptr(unsafe.Pointer(&mode))); r == 0 {
		say("FAIL: CONIN$ is not a console handle")
		os.Exit(1)
	}
	say("phase 0 OK: attached to the child's console (mode=0x%X)", mode)

	if r, _, _ = pSetScreenSize.Call(hOut,
		packCoord(int16(constCols), int16(constRows))); r == 0 {
		say("note: SetConsoleScreenBufferSize failed, using the default size")
	}
	var reg [4]int16
	reg[2], reg[3] = int16(constCols-1), int16(constRows-1)
	if r, _, _ = pSetWindowInfo.Call(hOut, 1,
		uintptr(unsafe.Pointer(&reg[0]))); r == 0 {
		say("note: SetConsoleWindowInfo failed, using the default window")
	}

	s := &sess{hIn: hIn, hOut: hOut, cols: constCols, rows: constRows, log: log}

	// ---- phase 1: the child must actually open the picker ----------------
	found := s.wait("the picker to render", 180*time.Second, func() bool {
		t := s.text()
		return strings.Contains(t, "序号") && strings.Contains(t, "音质")
	})
	if !found {
		s.dump("no picker")
		os.Exit(1)
	}
	say("phase 1 OK: the picker rendered inside the console")
	s.dump("picker open")

	// ---- phase 2: one arrow-down must move exactly one row ---------------
	hl := highlightRows(s.screen())
	say("phase 2: highlighted rows before = %v", hl)
	if len(hl) != 1 {
		kill("expected exactly one highlighted row before, got %v", hl)
	}
	before := hl[0]

	s.press(vkDown, 0, 0)
	moved := s.wait("one row of movement", 5*time.Second, func() bool {
		h := highlightRows(s.screen())
		return len(h) == 1 && h[0] != before
	})
	if !moved {
		s.dump("after down")
		kill("a single arrow-down did not move the selection")
	} else {
		h := highlightRows(s.screen())
		if h[0] == before+1 {
			say("phase 2 OK: one arrow-down moved exactly one row (%d -> %d)", before, h[0])
		} else {
			s.dump("after down")
			kill("arrow-down moved the selection to row %d, want %d", h[0], before+1)
		}
	}

	// ---- phase 3: PgDn must jump a whole page, not one row ---------------
	b2 := highlightRows(s.screen())
	start2 := b2[0]
	s.press(vkNext, 0, enhanced)
	if s.wait("a page jump", 5*time.Second, func() bool {
		h := highlightRows(s.screen())
		return len(h) == 1 && h[0] > start2
	}) {
		h := highlightRows(s.screen())
		if h[0] > start2+1 {
			say("phase 3 OK: PgDn jumped %d rows (%d -> %d)", h[0]-start2, start2, h[0])
		} else {
			say("note: PgDn moved one row (%d -> %d); the list may already fit", start2, h[0])
		}
	} else {
		s.dump("after PgDn")
		kill("PgDn did not move the selection")
	}

	// ---- phase 4: Home back to the first data row ------------------------
	if s.wait("Home", 5*time.Second, func() bool {
		s.press(vkHome, 0, enhanced)
		h := highlightRows(s.screen())
		return len(h) == 1 && h[0] == before
	}) {
		say("phase 4 OK: Home returned the selection to row %d", before)
	} else {
		s.dump("after Home")
		kill("Home did not return to the first row")
	}

	// ---- phase 5: Enter shows the confirmation, it does not download -----
	s.press(vkReturn, vkReturn, 0)
	showed := s.wait("the confirmation prompt", 5*time.Second, func() bool {
		t := s.text()
		return strings.Contains(t, "[y]") && strings.Contains(t, "[n]")
	})
	if !showed {
		s.dump("after Enter")
		kill("Enter did not show the confirmation prompt")
	} else {
		s.dump("confirmation prompt")
		say("phase 5 OK: Enter showed the confirmation prompt")
	}

	// ---- phase 6: 'n' cancels back to the table --------------------------
	s.press(0, 'n', 0)
	if s.wait("the prompt to clear", 5*time.Second, func() bool {
		return !strings.Contains(s.text(), "[y]")
	}) {
		say("phase 6 OK: 'n' dismissed the prompt and returned to the table")
	} else {
		s.dump("after n")
		kill("'n' did not dismiss the prompt")
	}

	// ---- phase 7: Enter then 'y' downloads the row -----------------------
	s.press(vkReturn, vkReturn, 0)
	if !s.wait("the prompt again", 5*time.Second, func() bool {
		return strings.Contains(s.text(), "[y]")
	}) {
		s.dump("second Enter")
		kill("the second Enter did not show the prompt")
	}
	s.press(0, 'y', 0)
	did := s.wait("the download to finish", 120*time.Second, func() bool {
		t := s.text()
		return strings.Contains(t, "已保存") || strings.Contains(t, "已下载")
	})
	if !did {
		s.dump("after y")
		kill("the download did not complete")
	} else {
		s.dump("download done")
		say("phase 7 OK: Enter -> y downloaded the selected row")
	}

	// ---- phase 8: 'q' closes the picker and exits cleanly ---------------
	s.press(0, 'q', 0)
	var w uintptr
	w, _, _ = pWaitFor.Call(pi[0], 30000)
	if w != 0 {
		s.dump("after q")
		kill("'q' did not quit the picker")
	}
	var code uint32
	pGetExitCode.Call(pi[0], uintptr(unsafe.Pointer(&code)))
	pWaitFor.Call(pi[1], 30000)
	pCloseHandle.Call(pi[0])
	pCloseHandle.Call(pi[1])
	say("phase 8: child exit code = %d", code)
	if code != 0 {
		kill("non-zero exit code %d", code)
	} else {
		say("phase 8 OK: 'q' quit the picker, exit code 0")
	}

	say("---")
	if fail {
		say("RESULT: FAILURES PRESENT")
	} else {
		say("RESULT: ALL PHASES PASSED")
	}
	if fail {
		os.Exit(1)
	}
}

func stringPtr(s string) *string {
	return &s
}
