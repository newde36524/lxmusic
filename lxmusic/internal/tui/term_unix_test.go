//go:build linux || darwin || freebsd || dragonfly || netbsd || openbsd

package tui

import (
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// TestNextKeyReadsStdin is the check for the carry-buffer spin. scan answers
// ok=true for an empty buffer, so a nextKey that loops on a keyless verdict
// never reaches the read and the picker answers to nothing. It redirects
// os.Stdin onto a pipe, so it needs no terminal and runs in CI.
func TestNextKeyReadsStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = old
		_ = r.Close()
		_ = w.Close()
	}()
	keyBuf = nil
	done := make(chan Key, 1)
	go func() {
		k, _ := nextKey()
		done <- k
	}()
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte{'q'})
	}()
	select {
	case k := <-done:
		if k != KeyQuit {
			t.Fatalf("nextKey returned %v, want KeyQuit", k)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("nextKey never read stdin: the keyless carry-buffer verdict spins before the read")
	}
}

// TestNextKeySurvivesEmptyRead is the check for reading through os.Stdin.
// Go's wrapper turns a zero-byte read into io.EOF, and with VMIN=0/VTIME=1 a
// read timeout IS a zero-byte read - so going through os.Stdin made nextKey
// report a fatal error a hundred milliseconds after the last keypress and
// killed the key reader, leaving a picker that was deaf to everything after.
// /dev/null never yields a byte and never errors at the syscall level, so the
// reader must keep waiting rather than returning.
func TestNextKeySurvivesEmptyRead(t *testing.T) {
	devnull, err := os.Open("/dev/null")
	if err != nil {
		t.Fatalf("devnull: %v", err)
	}
	defer devnull.Close()
	old := os.Stdin
	os.Stdin = devnull
	keyBuf = nil
	done := make(chan error, 1)
	go func() {
		_, e := nextKey()
		done <- e
	}()
	select {
	case e := <-done:
		t.Fatalf("nextKey returned on a zero-byte read: %v", e)
	case <-time.After(300 * time.Millisecond):
		// still waiting: a timeout is not an error
	}
	// Let the stray reader leave: closing the descriptor makes its next read
	// error, which is a genuine failure it is meant to report.
	_ = devnull.Close()
	<-done
	os.Stdin = old
}

// TestEnableRawMode is the guard for the two bugs that made the picker// unreachable on BSDs, both of which are invisible without a real terminal:
//
// enable() had to clear the flags it claimed to clear, and the values had to be
// the platform's own - ICANON and ISIG sit at 0x100 and 0x80 on every BSD and at
// 0x2 and 0x1 on Linux, so hardcoding one kernel's pair left input canonical on
// the others and a read blocked until the user hit Enter. A pipe cannot hold
// termios, so this skips where there is no tty, which is how CI runs it.
func TestEnableRawMode(t *testing.T) {
	fd := int(os.Stdin.Fd())
	var before syscall.Termios
	if err := ioctl(tcgets, fd, unsafe.Pointer(&before)); err != 0 {
		t.Skipf("stdin is not a terminal: %v", err)
	}
	if err := enable(); err != nil {
		t.Fatalf("enable: %v", err)
	}
	defer disable()

	var after syscall.Termios
	if err := ioctl(tcgets, fd, unsafe.Pointer(&after)); err != 0 {
		t.Fatalf("tcgets after enable: %v", err)
	}
	const flags = syscall.ECHO | syscall.ICANON | syscall.ISIG
	if got := after.Lflag & flags; got != 0 {
		t.Errorf("enable left %#x of ECHO|ICANON|ISIG set", got)
	}
	if got := after.Cc[syscall.VMIN]; got != 0 {
		t.Errorf("VMIN is %d, want 0", got)
	}
	if got := after.Cc[syscall.VTIME]; got != 1 {
		t.Errorf("VTIME is %d, want 1", got)
	}

	disable()
	var restored syscall.Termios
	if err := ioctl(tcgets, fd, unsafe.Pointer(&restored)); err != 0 {
		t.Fatalf("tcgets after disable: %v", err)
	}
	if got := restored.Lflag & flags; got != before.Lflag&flags {
		t.Errorf("disable did not put back the flags it cleared: Lflag %#x, want %#x",
			restored.Lflag, before.Lflag&flags)
	}
	if restored.Cc[syscall.VMIN] != before.Cc[syscall.VMIN] {
		t.Errorf("disable left VMIN %d, want %d", restored.Cc[syscall.VMIN], before.Cc[syscall.VMIN])
	}
}
