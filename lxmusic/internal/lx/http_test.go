package lx

import (
	"io"
	"strings"
	"testing"
)

// TestReportingReader pins the progress accounting the download bar relies on:
// the running count is cumulative and monotone, the announced total is passed
// through unchanged, and the final call carries the whole body.
func TestReportingReader(t *testing.T) {
	var calls [][2]int64
	r := &reportingReader{r: strings.NewReader("hello, world"), total: 12,
		on: func(done, total int64) { calls = append(calls, [2]int64{done, total}) }}
	all, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(all) != "hello, world" {
		t.Fatalf("body = %q, want the source bytes", all)
	}
	if len(calls) == 0 {
		t.Fatal("the callback was never called")
	}
	for i, c := range calls {
		if c[1] != 12 {
			t.Errorf("call %d announced total %d, want 12", i, c[1])
		}
		if i > 0 && c[0] < calls[i-1][0] {
			t.Errorf("call %d went backwards: %d after %d", i, c[0], calls[i-1][0])
		}
	}
	if got := calls[len(calls)-1][0]; got != 12 {
		t.Errorf("final count = %d, want 12", got)
	}
}

// TestReportingReaderNoLength covers the chunked / no Content-Length case, where
// the bar has to show a plain byte count instead of a percentage.
func TestReportingReaderNoLength(t *testing.T) {
	var gotTotal, gotDone int64
	r := &reportingReader{r: strings.NewReader("abc"), total: -1,
		on: func(done, total int64) { gotDone, gotTotal = done, total }}
	if _, err := io.ReadAll(r); err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if gotDone != 3 {
		t.Errorf("final count = %d, want 3", gotDone)
	}
	if gotTotal != -1 {
		t.Errorf("total = %d, want the announced -1 to pass through", gotTotal)
	}
}
