// Command diag reports why AllocConsole refuses on this machine.
//
// It is a probe, not product code.
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	stdIn    = ^uintptr(0) - 9
	stdOut   = ^uintptr(0) - 10
	curProc  = ^uintptr(0) - 1
	tokenSes = 12

	attrProcGroupAff  = 0x000C0008
	attrParentProcess = 0x00040004
)

var (
	k32    = syscall.NewLazyDLL("kernel32.dll")
	u32    = syscall.NewLazyDLL("user32.dll")
	ntdll  = syscall.NewLazyDLL("ntdll.dll")
	advapi = syscall.NewLazyDLL("advapi32.dll")

	pAllocConsole         = k32.NewProc("AllocConsole")
	pGetStdHandle         = k32.NewProc("GetStdHandle")
	pGetConsoleMode       = k32.NewProc("GetConsoleMode")
	pGetCurrentThreadAttr = k32.NewProc("GetCurrentProcessThreadAttributeValue")
	pInitAttrList         = k32.NewProc("InitializeProcThreadAttributeList")
	pUpdateAttr           = k32.NewProc("UpdateProcThreadAttribute")
	pDelAttrList          = k32.NewProc("DeleteProcThreadAttributeList")
	pGetProcessWindowStn  = u32.NewProc("GetProcessWindowStation")
	pOpenProcToken        = advapi.NewProc("OpenProcessToken")
	pGetTokenInfo         = advapi.NewProc("GetTokenInformation")
	pLookupPriv           = advapi.NewProc("LookupPrivilegeValueW")
	pGetTokenPrivileges   = advapi.NewProc("GetTokenPrivileges")
	pRtlGetVersion        = ntdll.NewProc("RtlGetVersion")
)

// call is LazyProc.Call without the panic: a missing export reports instead of
// aborting the whole probe.
func call(p *syscall.LazyProc, args ...uintptr) (uintptr, error) {
	if err := p.Find(); err != nil {
		return 0, err
	}
	r, _, e := p.Call(args...)
	return r, e
}

func main() {
	var ver [160]byte
	ver[0] = 160 // dwOSVersionInfoSize: OSVERSIONINFOEXW
	r, e := call(pRtlGetVersion, uintptr(unsafe.Pointer(&ver[0])))
	fmt.Printf("RtlGetVersion rc=%d err=%v -> build %d\n", r, e, le32(ver[:], 12))

	r, e = call(pAllocConsole)
	fmt.Printf("AllocConsole: rc=%d err=%v\n", r, e)

	for _, tc := range []struct {
		name string
		h    uintptr
	}{
		{"out", stdOut}, {"in", stdIn},
	} {
		h, _ := call(pGetStdHandle, tc.h)
		var mode uint32
		r, e := call(pGetConsoleMode, h, uintptr(unsafe.Pointer(&mode)))
		fmt.Printf("GetStdHandle(%s)=0x%X  GetConsoleMode rc=%d err=%v\n", tc.name, h, r, e)
	}

	r, e = call(pGetProcessWindowStn)
	fmt.Printf("GetProcessWindowStation=0x%X err=%v\n", r, e)

	r, e = call(pGetCurrentThreadAttr, uintptr(attrProcGroupAff))
	fmt.Printf("GetCurrentProcessThreadAttributeValue(PROCESS_GROUP_AFFINITY)=0x%X err=%v\n", r, e)

	var tok uintptr
	r, e = call(pOpenProcToken, curProc, 0x0008, uintptr(unsafe.Pointer(&tok)))
	fmt.Printf("OpenProcessToken rc=%d err=%v\n", r, e)
	if r != 0 {
		var sesid, need uint32
		r, _ = call(pGetTokenInfo, tok, tokenSes,
			uintptr(unsafe.Pointer(&sesid)), 4, uintptr(unsafe.Pointer(&need)))
		fmt.Printf("TokenSessionId=%d rc=%d\n", sesid, r)

		priv, _ := syscall.UTF16PtrFromString("SeCreateProcessGroupPrivilege")
		var luid [8]byte
		r, e = call(pLookupPriv, 0, uintptr(unsafe.Pointer(priv)),
			uintptr(unsafe.Pointer(&luid[0])))
		fmt.Printf("LookupPrivilegeValue rc=%d err=%v -> luid {%d, 0x%08X}\n",
			r, e, le32(luid[:], 0), le32(luid[:], 4))
		buf := make([]byte, 4+64*16)
		var cnt uint32
		r, _ = call(pGetTokenPrivileges, tok, uintptr(unsafe.Pointer(&cnt)),
			uintptr(unsafe.Pointer(&buf[0])))
		fmt.Printf("GetTokenPrivileges rc=%d count=%d\n", r, cnt)
		have := false
		for i := uint32(0); i < cnt && i < 64; i++ {
			b := int(i) * 16
			if le32(buf, b) == le32(luid[:], 0) && le32(buf, b+4) == le32(luid[:], 4) {
				have = true
				fmt.Printf("  -> present, attributes=0x%X\n", le32(buf, b+8))
			}
		}
		if !have {
			fmt.Printf("  -> SeCreateProcessGroupPrivilege is NOT in the token\n")
		}
	}

	var attrSize uintptr
	call(pInitAttrList, 0, 1, 0, uintptr(unsafe.Pointer(&attrSize)))
	fmt.Printf("attribute list size = %d\n", attrSize)
	buf := make([]byte, int(attrSize))
	r, e = call(pInitAttrList, uintptr(unsafe.Pointer(&buf[0])), 1, 0,
		uintptr(unsafe.Pointer(&attrSize)))
	fmt.Printf("InitializeProcThreadAttributeList rc=%d err=%v\n", r, e)

	unset := ^uintptr(0)
	for _, tc := range []struct {
		name string
		attr uintptr
	}{
		{"PROCESS_GROUP_AFFINITY=UNSET", attrProcGroupAff},
		{"PARENT_PROCESS=0", attrParentProcess},
	} {
		r, e = call(pUpdateAttr, uintptr(unsafe.Pointer(&buf[0])), 0, tc.attr,
			uintptr(unsafe.Pointer(&unset)), 8, 0, 0)
		fmt.Printf("UpdateProcThreadAttribute(%-28s) rc=%d err=%v\n", tc.name, r, e)
	}
	pDelAttrList.Call(uintptr(unsafe.Pointer(&buf[0])))
}

func le32(b []byte, off int) uint32 {
	return uint32(b[off]) | uint32(b[off+1])<<8 | uint32(b[off+2])<<16 | uint32(b[off+3])<<24
}
