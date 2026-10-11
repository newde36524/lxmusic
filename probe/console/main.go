// Command consoleprobe drives the real lxmusic.exe in a real Windows console.
//
// The harness has no console, so AllocConsole gives the parent its own, and the
// child — the unmodified lxmusic.exe — is attached to it. The parent then sends
// keystrokes with WriteConsoleInputW and reads back the screen with
// ReadConsoleOutputW, so the selection is verified from the console driver's
// own reverse-video attribute rather than by guessing at escape bytes.
//
// It is a probe, not product code. Nothing in the project imports it.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	stdOutput = ^uintptr(0) - 10 // STD_OUTPUT_HANDLE == -11
	stdInput  = ^uintptr(0) - 9  // STD_INPUT_HANDLE == -10

	enableVT           = 0x0004
	keyEvent           = 1
	reverseVideoAttr   = 0x4000
	infinite           = uint32(0xFFFFFFFF)
	createNoWindowFlag = 0x08000000

	// INPUT_RECORD is 28 bytes. Field offsets as the running product reads
	// them, and as a console round-trip confirmed.
	offEventType = 0
	offKeyDown   = 4
	offVK        = 10
	offUChar     = 14
	offKeyState  = 16

	vkUp     = 0x26
	vkDown   = 0x28
	vkReturn = 0x0D

	screenCols = 220
	screenRows = 60
)

var k32 = syscall.NewLazyDLL("kernel32.dll")

var (
	pAllocConsole   = k32.NewProc("AllocConsole")
	pFreeConsole    = k32.NewProc("FreeConsole")
	pGetStdHandle   = k32.NewProc("GetStdHandle")
	pGetConsoleMode = k32.NewProc("GetConsoleMode")
	pSetConsoleMode = k32.NewProc("SetConsoleMode")
	pWriteInput     = k32.NewProc("WriteConsoleInputW")
	pReadOutput     = k32.NewProc("ReadConsoleOutputW")
)

type smallRect struct{ left, top, right, bottom int16 }

// keyEventRecord is a KEY_EVENT_RECORD inside an INPUT_RECORD, written at the
// offsets the product's reader expects.
func makeKeyEvent(vk uint16, ch uint16) []byte {
	rec := make([]byte, 28)
	le16(rec, offEventType, keyEvent)
	le16(rec, offKeyDown, 1)
	le16(rec, offVK, vk)
	le16(rec, offUChar, ch)
	le32(rec, offKeyState, 0)
	return rec
}

func le16(b []byte, off int, v uint16) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
}

func le32(b []byte, off int, v uint32) {
	b[off] = byte(v)
	b[off+1] = byte(v >> 8)
	b[off+2] = byte(v >> 16)
	b[off+3] = byte(v >> 24)
}

func sendKeys(h uintptr, rec []byte) {
	var n uint32
	pWriteInput.Call(h, uintptr(unsafe.Pointer(&rec[0])), 1,
		uintptr(unsafe.Pointer(&n)))
}

// screen reads the visible buffer back out of the console driver.
func screen(h uintptr) (text []string, rev []int) {
	buf := make([]byte, screenCols*screenRows*4)
	reg := smallRect{0, 0, screenCols - 1, screenRows - 1}
	dim := [2]int16{screenCols, screenRows}
	n := int16(0)
	if r, _, _ := pReadOutput.Call(h, uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&dim[0])), uintptr(unsafe.Pointer(&reg)),
		uintptr(unsafe.Pointer(&n))); r == 0 {
		return nil, nil
	}
	text = make([]string, 0, screenRows)
	rev = nil
	for r := 0; r < screenRows; r++ {
		var sb strings.Builder
		for c := 0; c < screenCols; c++ {
			i := (r*screenCols + c) * 4
			rune16 := uint16(buf[i]) | uint16(buf[i+1])<<8
			attr := uint16(buf[i+2]) | uint16(buf[i+3])<<8
			if rune16 == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteString(string(utf16.Decode([]uint16{rune16})))
			}
			if attr&reverseVideoAttr != 0 {
				if len(rev) == 0 || rev[len(rev)-1] != r {
					rev = append(rev, r)
				}
			}
		}
		text = append(text, sb.String())
	}
	return text, rev
}

func containsText(text []string, s string) bool {
	for _, ln := range text {
		if strings.Contains(ln, s) {
			return true
		}
	}
	return false
}

// waitFor polls until fn reports true or the deadline passes.
func waitFor(what string, d time.Duration, fn func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	fmt.Printf("TIMEOUT waiting for %s\n", what)
	return false
}

func die(msg string) {
	fmt.Println("FAIL: " + msg)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: consoleprobe <lxmusic.exe> <keyword> <outDir>")
		os.Exit(2)
	}
	binary, keyword, outDir := os.Args[1], os.Args[2], os.Args[3]

	// 1. Give this process its own console so nothing else can read the traffic.
	if r, _, _ := pAllocConsole.Call(); r == 0 {
		die("AllocConsole failed")
	}
	defer pFreeConsole.Call()
	outH, _, _ := pGetStdHandle.Call(stdOutput)
	inH, _, _ := pGetStdHandle.Call(stdInput)
	if outH == 0 || inH == 0 {
		die("no console standard handles")
	}
	var mode uint32
	pGetConsoleMode.Call(outH, uintptr(unsafe.Pointer(&mode)))
	pSetConsoleMode.Call(outH, uintptr(mode|enableVT))

	// 2. Start the real binary attached to this console.
	cmd := exec.Command(binary, "search", keyword, "--source", "kg", "--limit", "10", "--out", outDir)
	cmd.Stdin = os.NewFile(inH, "console-in")
	cmd.Stdout = os.NewFile(outH, "console-out")
	cmd.Stderr = os.NewFile(outH, "console-out")
	if err := cmd.Start(); err != nil {
		die("start: " + err.Error())
	}
	fmt.Println("phase 0 OK: lxmusic.exe running in a real console")

	// 3. Wait for the picker to render.
	if !waitFor("the picker to render", 120*time.Second, func() bool {
		t, _ := screen(outH)
		return containsText(t, "第 ")
	}) {
		t, _ := screen(outH)
		fmt.Println("--- screen at timeout ---")
		for i, ln := range t {
			if strings.TrimSpace(ln) != "" {
				fmt.Printf("%2d|%s|\n", i+1, ln)
			}
		}
		cmd.Process.Kill()
		os.Exit(1)
	}
	t, rev := screen(outH)
	fmt.Printf("phase 1 OK: picker rendered; reverse-video rows = %v\n", rev)
	for i, ln := range t {
		if strings.TrimSpace(ln) != "" && i < 12 {
			fmt.Printf("%2d|%s|\n", i+1, ln)
		}
	}

	// 4. Two down arrows: the selection must move from the first data row to
	// the third.
	rev0 := rev
	for i := 0; i < 2; i++ {
		sendKeys(inH, makeKeyEvent(vkDown, 0))
	}
	time.Sleep(500 * time.Millisecond)
	_, rev1 := screen(outH)
	fmt.Printf("phase 2: selected rows before=%v after=%v\n", rev0, rev1)
	if len(rev0) < 1 || len(rev1) < 1 || rev1[len(rev1)-1] <= rev0[0] {
		die("the down arrow did not move the selection down")
	}
	fmt.Printf("phase 2 OK: arrow keys move the selection (row %d -> %d)\n", rev0[0], rev1[len(rev1)-1])

	// 5. Enter downloads the selected row.
	sendKeys(inH, makeKeyEvent(vkReturn, vkReturn))
	if !waitFor("the download to finish", 240*time.Second, func() bool {
		t, _ := screen(outH)
		return containsText(t, "已保存") || containsText(t, "✗")
	}) {
		t, _ := screen(outH)
		fmt.Println("--- screen at timeout ---")
		for i, ln := range t {
			if strings.TrimSpace(ln) != "" {
				fmt.Printf("%2d|%s|\n", i+1, ln)
			}
		}
		cmd.Process.Kill()
		os.Exit(1)
	}
	fmt.Println("phase 3 OK: Enter triggered the download and it completed")

	// 6. 'q' must close the picker and the process must exit.
	sendKeys(inH, makeKeyEvent(0x71, 'q'))
	if err := cmd.Wait(); err != nil {
		die("child exited with error: " + err.Error())
	}
	fmt.Println("phase 4 OK: 'q' quit the picker and the child exited cleanly")

	// 7. The file really landed on disk.
	entries, err := os.ReadDir(outDir)
	if err != nil || len(entries) == 0 {
		die(fmt.Sprintf("no file was written to %s (%v)", outDir, err))
	}
	fmt.Printf("phase 5 OK: %d file(s) on disk\n", len(entries))
	for _, e := range entries {
		if fi, err := e.Info(); err == nil {
			fmt.Printf("  %s  %.2f MB\n", e.Name(), float64(fi.Size())/1048576)
		}
	}

	// 8. Structural checks against the final screen.
	t2, _ := screen(outH)
	flat := strings.Join(t2, "\n")
	checks := map[string]string{
		"header 序号":         "序号",
		"header 平台":         "平台",
		"header 音乐名称":       "音乐名称",
		"header 时长":         "时长",
		"header 作者":         "作者",
		"header 音质":         "音质",
		"table separator":   "│",
		"footer help":       "q/Esc",
		"downloaded status": "已保存",
	}
	bad := 0
	for name, want := range checks {
		if !strings.Contains(flat, want) {
			fmt.Printf("MISSING: %s (%q)\n", name, want)
			bad++
		}
	}
	if bad != 0 {
		die(fmt.Sprintf("%d check(s) failed", bad))
	}
	fmt.Printf("phase 6 OK: all %d screen checks passed\n", len(checks))
	fmt.Println("ALL PHASES PASSED")
}
