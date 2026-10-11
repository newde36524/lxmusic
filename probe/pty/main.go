// Command pty runs lxmusic.exe inside a real Windows pseudoconsole, feeds it
// keystrokes, and reports what the picker actually did. It exists because a
// picker can only be proven on a terminal: the harness has no console, so this
// hands a pseudoconsole to the child instead.
//
// It is a probe, not product code. Nothing in the project imports it.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	attrListFlag       = 0x0001
	attrRestrictedFlag = 0x0002
	createNoWindowFlag = 0x08000000
	attrPseudoconsole  = 0x00000002
	handleInheritFlag  = 0x00000002

	esc         = "\x1b"
	outClear    = esc + "[2K"
	outRev      = esc + "[7m"
	outClearAll = esc + "[J"
)

type coord struct{ x, y int16 }

// procAttr is PROCESS_ATTRIBUTES.
type procAttr struct {
	size    uint32
	flags   uint32
	rootDir uintptr
	handles *uintptr
}

// startupInfoEx is STARTUPINFOEXW. Its first 112 bytes are STARTUPINFO, then a
// SIZE_T cbAttributeList, then the attribute list pointer at offset 120. Only
// those two fields are ever written, so the middle is opaque padding: no
// field-level knowledge of STARTUPINFO is needed or can drift.
type startupInfoEx struct {
	pad      [120]byte
	attrList *byte
}

type procInfo struct {
	process uintptr
	thread  uintptr
	pid     uint32
	tid     uint32
}

var (
	k32            = syscall.NewLazyDLL("kernel32.dll")
	pCreatePipe    = k32.NewProc("CreatePipe")
	pSetHandleInfo = k32.NewProc("SetHandleInformation")
	pCreateProcess = k32.NewProc("CreateProcessW")
	pWaitForObject = k32.NewProc("WaitForSingleObject")
	pGetExitCode   = k32.NewProc("GetExitCodeProcess")
	pCloseHandle   = k32.NewProc("CloseHandle")
	pInitAttrList  = k32.NewProc("InitializeProcThreadAttributeList")
	pUpdateAttr    = k32.NewProc("UpdateProcThreadAttribute")
	pDelAttrList   = k32.NewProc("DeleteProcThreadAttributeList")
	pWriteFile     = k32.NewProc("WriteFile")
	pReadFile      = k32.NewProc("ReadFile")

	ptyDll     = syscall.NewLazyDLL("api-ms-win-console-pty-l1-1-0.dll")
	pCreatePty = ptyDll.NewProc("CreatePseudoConsole")
)

// capture is the PTY output as one growing string, safe to read while the
// reader goroutine writes.
type capture struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (c *capture) add(s string) {
	c.mu.Lock()
	c.buf.WriteString(s)
	c.mu.Unlock()
}

func (c *capture) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// waitFor polls until fn reports true or the deadline passes.
func waitFor(what string, d time.Duration, fn func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Printf("TIMEOUT waiting for %s\n", what)
	return false
}

// writeTo pushes bytes into the pseudoconsole's input pipe.
func writeTo(h uintptr, s string) {
	b := []byte(s)
	var n uint32
	pWriteFile.Call(h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)),
		uintptr(unsafe.Pointer(&n)), 0)
}

// selectedRows scans the captured traffic for "\x1b[N;1H\x1b[2K\x1b[7m..." and
// returns the line numbers the picker painted in reverse video, i.e. the rows
// it actually selected.
func selectedRows(out string) []int {
	set := map[int]bool{}
	i := 0
	for i < len(out) {
		if out[i] != 0x1b {
			i++
			continue
		}
		rest := out[i+1:]
		j := strings.Index(rest, ";1H")
		if j < 0 {
			i++
			continue
		}
		line, err := strconv.Atoi(rest[:j])
		after := rest[j+3:]
		if err != nil || !strings.HasPrefix(after, outClear) {
			i += j + 3
			continue
		}
		body := after[len(outClear):]
		if k := strings.Index(body, "\x1b"); k >= 0 {
			body = body[:k]
		} else if k := strings.Index(body, outClearAll); k >= 0 {
			body = body[:k]
		} else if k := strings.Index(body, "\r\n"); k >= 0 {
			body = body[:k]
		}
		if strings.HasPrefix(body, outRev) {
			set[line] = true
		}
		i += j + 3
	}
	keys := []int{}
	for k := range set {
		keys = append(keys, k)
	}
	for a := 0; a < len(keys); a++ {
		for b := a + 1; b < len(keys); b++ {
			if keys[a] > keys[b] {
				keys[a], keys[b] = keys[b], keys[a]
			}
		}
	}
	return keys
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: pty <lxmusic.exe> <keyword> <outDir>")
		os.Exit(2)
	}
	binary, keyword, outDir := os.Args[1], os.Args[2], os.Args[3]
	const cols, rows = int16(120), int16(30)

	fail := func(format string, a ...any) {
		fmt.Printf("FAIL: "+format+"\n", a...)
		os.Exit(1)
	}

	// Two pipes: the child reads inRead (we write inWrite) and writes
	// outWrite (we read outRead). Stderr goes to outWrite too, so any error the
	// child reports lands in the same capture.
	var inRead, inWrite, outRead, outWrite uintptr
	for _, pair := range [][]*uintptr{{&inRead, &inWrite}, {&outRead, &outWrite}} {
		r, _, err := pCreatePipe.Call(uintptr(unsafe.Pointer(pair[0])),
			uintptr(unsafe.Pointer(pair[1])))
		if r == 0 {
			fail("CreatePipe: %v", err)
		}
	}
	for _, h := range []uintptr{inRead, inWrite, outRead, outWrite} {
		pSetHandleInfo.Call(h, handleInheritFlag, handleInheritFlag)
	}

	var pc uintptr
	size := coord{cols, rows}
	if r, _, _ := pCreatePty.Call(uintptr(unsafe.Pointer(&size)), inRead, outWrite, 0,
		uintptr(unsafe.Pointer(&pc))); r != 0 {
		fail("CreatePseudoConsole HRESULT=0x%X", int32(r))
	}

	var attrSize uintptr
	if r, _, _ := pInitAttrList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrSize))); r == 0 || attrSize == 0 {
		fail("InitializeProcThreadAttributeList size probe")
	}
	attrBuf := make([]byte, int(attrSize))
	if r, _, err := pInitAttrList.Call(uintptr(unsafe.Pointer(&attrBuf[0])), 1, 0,
		uintptr(unsafe.Pointer(&attrSize))); r == 0 {
		fail("InitializeProcThreadAttributeList: %v", err)
	}
	defer pDelAttrList.Call(uintptr(unsafe.Pointer(&attrBuf[0])))
	if r, _, err := pUpdateAttr.Call(uintptr(unsafe.Pointer(&attrBuf[0])), 0,
		uintptr(attrPseudoconsole), uintptr(unsafe.Pointer(&pc)), 8, 0, 0); r == 0 {
		fail("UpdateProcThreadAttribute PSEUDOCONSOLE: %v", err)
	}

	handles := []uintptr{pc}
	pa := procAttr{size: 24, flags: attrListFlag | attrRestrictedFlag, handles: &handles[0]}

	cmdLine := binary + ` search ` + keyword + ` --source kg --limit 10 --out ` + outDir
	lp, err := syscall.UTF16PtrFromString(cmdLine)
	if err != nil {
		fail("utf16: %v", err)
	}

	var si startupInfoEx
	const siSize = 128
	si.pad[0], si.pad[1] = byte(siSize), byte(siSize>>8)
	si.attrList = (*byte)(unsafe.Pointer(&attrBuf[0]))
	var pi procInfo
	if r, _, err := pCreateProcess.Call(0, uintptr(unsafe.Pointer(lp)),
		uintptr(unsafe.Pointer(&pa)), 0, 1, uintptr(createNoWindowFlag), 0, 0,
		uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi))); r == 0 {
		fail("CreateProcessW: %v", err)
	}
	// The child owns those two pipe ends now.
	pCloseHandle.Call(inRead)
	pCloseHandle.Call(outWrite)

	out := &capture{}
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		var b [8192]byte
		for {
			var n uint32
			r, _, _ := pReadFile.Call(outRead, uintptr(unsafe.Pointer(&b[0])),
				uintptr(len(b)), uintptr(unsafe.Pointer(&n)), 0)
			if r == 0 || n == 0 {
				return
			}
			out.add(string(b[:n]))
			if w, _, _ := pWaitForObject.Call(pi.process, 50); w == 0 {
				var m uint32
				r2, _, _ := pReadFile.Call(outRead, uintptr(unsafe.Pointer(&b[0])),
					uintptr(len(b)), uintptr(unsafe.Pointer(&m)), 0)
				if r2 != 0 && m != 0 {
					out.add(string(b[:m]))
				}
				return
			}
		}
	}()

	// Phase 1: the search runs over the network, then the picker renders.
	if !waitFor("the picker to render", 120*time.Second, func() bool {
		return strings.Contains(out.get(), "第 ")
	}) {
		fmt.Printf("--- captured %d bytes ---\n%s\n", len(out.get()), out.get())
		os.Exit(1)
	}
	fmt.Println("phase 1 OK: picker rendered inside the pseudoconsole")

	// Phase 2: two down arrows. The selection must move from the first data
	// row (line 4) to the third (line 6).
	before := selectedRows(out.get())
	writeTo(inWrite, esc+"[B"+esc+"[B") // down, down
	time.Sleep(600 * time.Millisecond)
	after := selectedRows(out.get())
	fmt.Printf("phase 2: selected lines before=%v after=%v\n", before, after)
	moved := false
	for _, l := range after {
		if l == 6 {
			moved = true
		}
	}
	if !moved {
		fmt.Println("FAIL: the down arrow did not move the selection to the third row")
		os.Exit(1)
	}
	fmt.Println("phase 2 OK: arrow keys move the selection (line 4 -> line 6)")

	// Phase 3: Enter downloads the selected row.
	writeTo(inWrite, "\r")
	if !waitFor("the download to finish", 180*time.Second, func() bool {
		s := out.get()
		return strings.Contains(s, "已保存") || strings.Contains(s, "✗")
	}) {
		fmt.Printf("--- captured %d bytes ---\n%s\n", len(out.get()), out.get())
		os.Exit(1)
	}
	fmt.Println("phase 3 OK: Enter triggered the download and it completed")

	// Phase 4: q must close the picker and the process must exit cleanly.
	writeTo(inWrite, "q")
	if w, _, _ := pWaitForObject.Call(pi.process, 30000); w != 0 {
		fmt.Println("FAIL: the picker did not quit on 'q'")
		os.Exit(1)
	}
	var code uint32
	pGetExitCode.Call(pi.process, uintptr(unsafe.Pointer(&code)))
	pWaitForObject.Call(pi.thread, 30000)
	pCloseHandle.Call(pi.process)
	pCloseHandle.Call(pi.thread)
	<-drained

	fmt.Printf("phase 4 OK: 'q' quit the picker, child exit code = %d\n", code)
	if code != 0 {
		fmt.Println("FAIL: non-zero exit code")
		os.Exit(1)
	}

	// Phase 5: the file really landed on disk.
	entries, err := os.ReadDir(outDir)
	if err != nil || len(entries) == 0 {
		fmt.Printf("FAIL: no file was written to %s (%v)\n", outDir, err)
		os.Exit(1)
	}
	fmt.Printf("phase 5 OK: %d file(s) on disk\n", len(entries))
	for _, e := range entries {
		if fi, err := e.Info(); err == nil {
			fmt.Printf("  %s  %.2f MB\n", e.Name(), float64(fi.Size())/1048576)
		}
	}

	// Phase 6: structural checks on what the picker actually painted.
	s := out.get()
	checks := map[string]string{
		"header 序号":          "序号",
		"header 平台":          "平台",
		"header 音乐名称":        "音乐名称",
		"header 时长":          "时长",
		"header 作者":          "作者",
		"header 音质":          "音质",
		"alt screen entered": esc + "[?1049h",
		"alt screen left":    esc + "[?1049l",
		"reverse-video row":  outRev,
		"clear line":         outClear,
		"clear to end":       outClearAll,
		"cursor column move": ";1H" + outClear,
		"downloaded status":  "已保存",
	}
	bad := 0
	for name, want := range checks {
		if !strings.Contains(s, want) {
			fmt.Printf("MISSING: %s (%q)\n", name, want)
			bad++
		}
	}
	if strings.Contains(s, "error:") {
		bad++
		for _, ln := range strings.Split(s, "\r\n") {
			if strings.Contains(ln, "error:") {
				fmt.Println("child stderr: " + strings.TrimSpace(ln))
			}
		}
	}
	if bad != 0 {
		fmt.Printf("phase 6: %d check(s) failed\n", bad)
		os.Exit(1)
	}
	fmt.Printf("phase 6 OK: all %d render checks passed (%d bytes of terminal traffic)\n",
		len(checks), len(s))
	fmt.Println("ALL PHASES PASSED")
}
