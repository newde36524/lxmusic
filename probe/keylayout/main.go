// Determine whether WriteConsoleInputW/ReadConsoleInputW memcpy the record or
// re-serialize it. If memcpy, the struct layout is unknowable from the API and
// must be taken from the SDK. Safe: one fixed 256-byte buffer, never undersized.
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
	GENERIC_READ  = 0x80000000
	GENERIC_WRITE = 0x40000000
	SYNCHRONIZE   = 0x00100000
	KEY_EVENT     = 1
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

func main() {
	name, _ := syscall.UTF16PtrFromString("CONIN$")
	h, _, _ := pCreateFileW.Call(uintptr(unsafe.Pointer(name)),
		uintptr(GENERIC_READ|GENERIC_WRITE|SYNCHRONIZE), 0, 0, 3, 0, 0)
	if h == 0 {
		panic("no CONIN$")
	}
	defer pCloseHandle.Call(h)
	fmt.Printf("uintptr=%d bytes, CONIN$ handle=%d\n\n", unsafe.Sizeof(uintptr(0)), h)

	fmt.Println("TEST 1: markers in the struct padding and the MOUSE-union tail")
	pFlushConsoleInput.Call(h)
	src := make([]byte, 256)
	src[0] = KEY_EVENT
	src[2], src[3], src[4], src[5], src[6], src[7] = 0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56
	src[24], src[25], src[26], src[27] = 0x77, 0x88, 0x99, 0xAA
	var w, n uint32
	wr, _, _ := pWriteConsoleInputW.Call(h, uintptr(unsafe.Pointer(&src[0])), 1,
		uintptr(unsafe.Pointer(&w)))
	dst := make([]byte, 256)
	rd, _, _ := pReadConsoleInputW.Call(h, uintptr(unsafe.Pointer(&dst[0])), 1,
		uintptr(unsafe.Pointer(&n)))
	fmt.Printf("  write=%d(%d) read=%d(%d)\n", wr, w, rd, n)
	fmt.Printf("  echo : %s\n\n", dump(dst))

	fmt.Println("TEST 2: VK_UP written ONLY at SDK offset 14 (offset 12 left zero)")
	pFlushConsoleInput.Call(h)
	src = make([]byte, 256)
	src[0] = KEY_EVENT
	src[8] = 1  // bKeyDown
	src[10] = 1 // wRepeatCount
	src[14], src[15] = 0x26, 0
	wr, _, _ = pWriteConsoleInputW.Call(h, uintptr(unsafe.Pointer(&src[0])), 1,
		uintptr(unsafe.Pointer(&w)))
	dst = make([]byte, 256)
	rd, _, _ = pReadConsoleInputW.Call(h, uintptr(unsafe.Pointer(&dst[0])), 1,
		uintptr(unsafe.Pointer(&n)))
	fmt.Printf("  write=%d read=%d\n", wr, rd)
	fmt.Printf("  echo : %s\n", dump(dst))
	for off := 0; off+2 <= len(dst); off++ {
		if binary.LittleEndian.Uint16(dst[off:off+2]) == 0x26 {
			fmt.Printf("  -> VK_UP observed at offset %d\n", off)
		}
	}
}
