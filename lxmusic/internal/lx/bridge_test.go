package lx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubSource is a user source that answers musicUrl with no network at all. The
// two environment knobs let a test stand in for a slow upstream: STUB_INIT_DELAY
// delays the lx.send(inited) call that a real script makes while it loads, and
// STUB_EXIT_FLAG makes the very first process quit before announcing itself, so
// the next one picks up where it left off.
const stubSource = `/*
 * @name StubSource
 * @version 1.0.0
 */
var fs = require("fs");
var flag = process.env.STUB_EXIT_FLAG;
if (flag && !fs.existsSync(flag)) {
  fs.writeFileSync(flag, "1");
  process.exit(7);
}
var n = 0;
var delay = Number(process.env.STUB_INIT_DELAY_MS || 0);
lx.on("request", function (req) {
  n++;
  return Promise.resolve("http://example.com/s" + n + ".mp3");
});
setTimeout(function () {
  lx.send("inited", {
    sources: { kw: { actions: ["musicUrl"], qualitys: ["128k", "320k"] } },
  });
}, delay);
`

// bridgeFixture writes the bridge and the stub to a temp dir and returns a
// source pointed at them. Node is required, because the bridge is node. When
// fast is set the per-request timeout is short, so a source that never answers
// fails in seconds instead of the default twenty-five.
func bridgeFixture(t *testing.T, fast bool, envs ...string) *UserSource {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on PATH")
	}
	dir := t.TempDir()
	// The script cache lands under APPDATA on Windows; point it here so the test
	// does not write into the user's own config.
	old, had := os.LookupEnv("APPDATA")
	if err := os.Setenv("APPDATA", dir); err != nil {
		t.Fatalf("setenv APPDATA: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("APPDATA", old)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
	})
	if err := writeBridge(dir); err != nil {
		t.Fatalf("writeBridge: %v", err)
	}
	script := filepath.Join(dir, "stub.js")
	if err := os.WriteFile(script, []byte(stubSource), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	for _, kv := range envs {
		k, _, _ := strings.Cut(kv, "=")
		if err := os.Setenv(k, kv[len(k)+1:]); err != nil {
			t.Fatalf("setenv: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, kv := range envs {
			k, _, _ := strings.Cut(kv, "=")
			_ = os.Unsetenv(k)
		}
	})
	sec := 15
	if fast {
		sec = 3
	}
	return &UserSource{
		C:    NewClient(),
		Spec: UserSourceSpec{Name: "stub", URL: script, TimeoutSec: sec},
	}
}

func stubCall(t *testing.T, u *UserSource) (any, string, error) {
	t.Helper()
	info := map[string]any{
		"source": "kw", "id": "1", "types": []string{"128k", "320k"}, "type": "320k",
		"musicInfo": map[string]any{"source": "kw", "id": "1", "type": "320k"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return u.call(ctx, "musicUrl", info)
}

// TestBridgeSessionReusesInit is the fix for the long pause before a download
// starts. A script's init used to run once per request because every request
// spawned a fresh node; with a session the second request goes straight to the
// handler.
func TestBridgeSessionReusesInit(t *testing.T) {
	u := bridgeFixture(t, false, "STUB_INIT_DELAY_MS=1500")

	t0 := time.Now()
	out, _, err := stubCall(t, u)
	first := time.Since(t0)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if s, _ := out.(string); !strings.HasPrefix(s, "http://example.com/s") {
		t.Fatalf("first call url = %v", out)
	}

	t0 = time.Now()
	out, _, err = stubCall(t, u)
	second := time.Since(t0)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if s, _ := out.(string); !strings.HasPrefix(s, "http://example.com/s") {
		t.Fatalf("second call url = %v", out)
	}

	// The first call waits out the init delay; the second must not. Wide margin
	// so a slow machine still passes - the assertion is "not paying it again",
	// not an exact number.
	if second > first/3 {
		t.Errorf("second call took %v after a first of %v; the init delay was paid twice", second, first)
	}
}

// TestBridgeSessionRespawnsAfterDeath covers a source that quits while it is
// loading, before it ever announces itself. The first process is lost and a
// fresh one has to take over: a wedged source would retry the dead session
// forever instead of ever answering.
func TestBridgeSessionRespawnsAfterDeath(t *testing.T) {
	u := bridgeFixture(t, true, "STUB_EXIT_FLAG="+filepath.Join(t.TempDir(), "died-once"))

	out, _, err := stubCall(t, u)
	if err != nil {
		t.Fatalf("did not recover from a process that died before init: %v", err)
	}
	if s, _ := out.(string); s == "" {
		t.Fatalf("recovered session returned no url")
	}
	if u.sess == nil {
		t.Fatalf("the recovered session was not kept")
	}

	// The survivor must then be reused, not re-forked on the next request.
	kept := u.sess
	_, _, err = stubCall(t, u)
	if err != nil {
		t.Fatalf("session reused after respawn failed: %v", err)
	}
	if u.sess != kept {
		t.Errorf("a healthy session was thrown away")
	}
}

// TestBridgeErrSinkBounds keeps the tail without growing without bound: a source
// that logs on every request would otherwise fill memory over a long session.
func TestBridgeErrSinkBounds(t *testing.T) {
	e := &errSink{}
	for i := 0; i < 50; i++ {
		if _, err := e.Write([]byte("line\n")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	got := e.tail()
	if len(got) > 4200 {
		t.Errorf("tail grew to %d bytes, want it bounded near the cap", len(got))
	}
	if !strings.Contains(got, "line") {
		t.Errorf("tail lost the recent content: %q", got)
	}
}
