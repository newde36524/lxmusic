package lx

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestTxSeedDump dumps the schedule seed and each rotation for key group 0,
// in the layout scripts/txseeddump.js writes.
func TestTxSeedDump(t *testing.T) {
	v := os.Getenv("TX_LYRIC_SEED")
	if v == "" {
		return
	}
	var g [8]byte
	copy(g[:], txKeyBytes[16:24])
	hi, lo := txSeed(g)
	lines := []string{fmt.Sprintf("seed %08x %08x", hi, lo)}
	for r := 0; r < 16; r++ {
		sh := txK[r]
		hi = (hi<<sh | hi>>(28-sh)) & 0xFFFFFFF0
		lo = (lo<<sh | lo>>(28-sh)) & 0xFFFFFFF0
		lines = append(lines, fmt.Sprintf("r%d %08x %08x", r, hi, lo))
	}
	if err := os.WriteFile(v, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTxxDump dumps txX(e,t,31-t) for t=0..63, the layout txxdump.js writes.
func TestTxxDump(t *testing.T) {
	v := os.Getenv("TX_LYRIC_X")
	if v == "" {
		return
	}
	var e [8]byte
	copy(e[:], txKeyBytes[16:24])
	lines := make([]string, 64)
	for t := 0; t < 64; t++ {
		lines[t] = fmt.Sprintf("%d %x", t, txX(e, uint(t), uint(31-t)))
	}
	if err := os.WriteFile(v, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTencentSubkeys dumps the three subkey matrices in the same layout as
// scripts/txkeydump.js so they can be diffed.
func TestTencentSubkeys(t *testing.T) {
	v := os.Getenv("TX_LYRIC_KEYDBG")
	if v == "" {
		return
	}
	var lines []string
	for i, k := range txSubkeys() {
		lines = append(lines, fmt.Sprintf("key %d", i))
		for r := 0; r < 16; r++ {
			b := make([]byte, 6)
			copy(b, k[r][:])
			lines = append(lines, hex.EncodeToString(b))
		}
		lines = append(lines, "")
	}
	if err := os.WriteFile(v, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The Go port must reproduce the bundle's decoder byte for byte. The reference
// file is produced by scripts/txjs.js running the transcribed cipher.
func TestTencentDecryptLyric(t *testing.T) {
	hexBlob, err := os.ReadFile(os.Getenv("TX_LYRIC_HEX"))
	if err != nil {
		t.Skip("TX_LYRIC_HEX not set")
	}
	ref, err := os.ReadFile(os.Getenv("TX_LYRIC_REF"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(hexBlob)))
	if err != nil {
		t.Fatal(err)
	}
	dec := txDecryptBlocks(append([]byte(nil), raw...))
	if v := os.Getenv("TX_LYRIC_DBG"); v != "" {
		if err := os.WriteFile(v, dec, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := TencentDecryptLyric(strings.TrimSpace(string(hexBlob)))
	want := strings.TrimSpace(string(ref))
	if got != want {
		t.Fatalf("mismatch:\n got %d chars, head %q\nwant %d chars, head %q",
			len(got), got[:min(len(got), 200)], len(want), want[:min(len(want), 200)])
	}
	for _, c := range []struct{ in, want string }{
		{"", ""}, {"abc", ""}, {"00", ""},
	} {
		if v := TencentDecryptLyric(c.in); v != c.want {
			t.Errorf("%q: got %q want %q", c.in, v, c.want)
		}
	}
}
