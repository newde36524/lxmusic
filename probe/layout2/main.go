// Pin down INPUT_RECORD's real byte layout for KEY_EVENT_RECORD on this host.
//
// The console driver re-serialises a key-DOWN record (TEST 1 of the earlier
// probe zeroed our padding markers and wrote 0x01 itself at offset 4), so the
// echo of a key-down is the driver telling us its own canonical layout. A
// key-UP is passed through as a memcpy, so it tells us nothing.
//
// Two candidate layouts are in play:
//   L2: EventHandle[0], bKeyDown[4], wRepeat[5], wVK[7],  wScan[9],  UChar[11], flags[13]
//   L8: EventHandle[0], pad[4],  bKeyDown[8], wRepeat[9], wVK[10], wScan[12], UChar[14], flags[16]
//
// Both are tried with the same values, so whichever set of offsets shows
// vk=0x28, scan=0x4E, uchar=0x41 in the echo is the real one.
package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"syscall"
)

var (
	k32                 = syscall.NewLazyDLL("kernel32.dll")
	pCreateFileW        = k32.NewProc("CreateFileW")
	pCloseHandle        = k32.NewProc("CloseHandle")
	pWriteConsoleInputW = k32.NewProc("WriteConsoleInputW")
	pReadConsoleInputW  = k32.NewProc("ReadConsoleInputW")
	pFlushConsoleInput  = k32.NewProc("FlushConsoleInputBuffer")
)

const (
	genericRW = 0x80000000 | 0x40000000 | 0x00100000
	keyEvent  = 1
	vkDown    = 0x28
	scanDown  = 0x4E
	ucharA    = 0x41
)

func dump(b []byte) string {
	s := ""
	for i, v := range b {
		if v != 0 {
			s += fmt.Sprintf("[%02d]=%02x ", i, v)
		}
	}
	if s == "" {
		return "(all zero)"
	}
	return s
}

func find16(b []byte, want uint16) []int {
	out := []int{}
	for off := 0; off+2 <= len(b); off++ {
		if binary.LittleEndian.Uint16(b[off:off+2]) == want {
			out = append(out, off)
		}
	}
	return out
}

// roundtrip writes one record, reads it back, and reports the echo.
func roundtrip(h uintptr, name string, down bool, layout string, src []byte) {
	pFlushConsoleInput.Call(h)
	var w, n uint32
	wr, _, _ := pWriteConsoleInputW.Call(h, uintptr(unsafe.Pointer(&src[0])), 1,
		uintptr(unsafe.Pointer(&w)))
	dst := make([]byte, 64)
	rd, _, _ := pReadConsoleInputW.Call(h, uintptr(unsafe.Pointer(&dst[0])), 1,
		uintptr(unsafe.Pointer(&n)))
	fmt.Printf("%-34s bKeyDown=%v write=%d read=%d\n", name, down, wr, rd)
	fmt.Printf("  echo: %s\n", dump(dst))
	fmt.Printf("  vk 0x28  at %v   scan 0x4E at %v   uchar 0x41 at %v\n\n",
		find16(dst, vkDown), find16(dst, scanDown), find16(dst, ucharA))
}

func main() {
	name, _ := syscall.UTF16PtrFromString("CONIN$")
	h, _, _ := pCreateFileW.Call(uintptr(unsafe.Pointer(name)),
		uintptr(genericRW), 0, 0, 3, 0, 0)
	if h == 0 {
		panic("no CONIN$")
	}
	defer pCloseHandle.Call(h)
	fmt.Printf("CONIN$ handle=%d\n\n", h)

	for _, layout := range []struct {
		name      string
		bk, rep   int
		vk, scan  int
		uc, flags int
	}{
		{"L2 (union @4)", 4, 5, 7, 9, 11, 13},
		{"L8 (union @8)", 8, 9, 10, 12, 14, 16},
	} {
		for _, down := range []bool{true, false} {
			src := make([]byte, 64)
			src[0] = keyEvent
			if down {
				src[layout.bk] = 1
			}
			src[layout.rep] = 1
			binary.LittleEndian.PutUint16(src[layout.vk:], vkDown)
			binary.LittleEndian.PutUint16(src[layout.scan:], scanDown)
			binary.LittleEndian.PutUint16(src[layout.uc:], ucharA)
			roundtrip(h, layout.name, down, layout.name, src)
		}
	}
}
