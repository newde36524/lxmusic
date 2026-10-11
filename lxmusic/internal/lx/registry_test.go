package lx

import "testing"

// withRegistry runs f with the package registry replaced, restoring whatever was
// there first. The real config file must never be touched by a test: these tests
// are about routing, not about the disk.
func withRegistry(entries map[string]Provider, f func()) {
	old := registry
	registry = entries
	defer func() { registry = old }()
	f()
}

func us(name, platform string) *UserSource {
	return &UserSource{Spec: UserSourceSpec{Name: name, URL: "https://example.invalid/s.js", Source: platform}}
}

// TestUserSourceFor pins the routing rule: a source pinned to a platform wins
// over one left general, and a pinned source never answers for a platform it
// was not pinned to.
func TestUserSourceFor(t *testing.T) {
	withRegistry(map[string]Provider{}, func() {
		if got := UserSourceFor("kw"); got != nil {
			t.Errorf("no sources configured, UserSourceFor(kw) = %v, want nil", got.ID())
		}
	})

	withRegistry(map[string]Provider{
		"hyw": us("hyw", ""),
	}, func() {
		// A general source answers for any platform.
		if got := UserSourceFor("kw"); got == nil || got.ID() != "hyw" {
			t.Errorf("UserSourceFor(kw) = %v, want hyw", idOrNil(got))
		}
		if got := UserSourceFor("mg"); got == nil || got.ID() != "hyw" {
			t.Errorf("UserSourceFor(mg) = %v, want hyw", idOrNil(got))
		}
		// No platform named: nothing to route on, so nothing to route to.
		if got := UserSourceFor(""); got != nil {
			t.Errorf("UserSourceFor(\"\") = %v, want nil", got.ID())
		}
		if got := UserSourceFor("   "); got != nil {
			t.Errorf("UserSourceFor(\"   \") = %v, want nil", got.ID())
		}
	})

	withRegistry(map[string]Provider{
		"hyw":    us("hyw", ""),
		"kwonly": us("kwonly", "kw"),
	}, func() {
		// The pinned source wins where it is pinned...
		if got := UserSourceFor("kw"); got == nil || got.ID() != "kwonly" {
			t.Errorf("UserSourceFor(kw) = %v, want kwonly", idOrNil(got))
		}
		// ...but the general one still covers everything else.
		if got := UserSourceFor("wy"); got == nil || got.ID() != "hyw" {
			t.Errorf("UserSourceFor(wy) = %v, want hyw", idOrNil(got))
		}
	})

	withRegistry(map[string]Provider{
		"kwonly": us("kwonly", "kw"),
	}, func() {
		// A pinned source answers for nothing outside its platform.
		if got := UserSourceFor("wy"); got != nil {
			t.Errorf("UserSourceFor(wy) = %v, want nil", got.ID())
		}
		// Case and whitespace in the hint must not matter.
		if got := UserSourceFor("KW "); got == nil || got.ID() != "kwonly" {
			t.Errorf("UserSourceFor(\"KW \") = %v, want kwonly", idOrNil(got))
		}
	})
}

// TestAudioProvider pins that a configured user source replaces the built-in
// answer, and that a user source asked for directly is not second-guessed.
func TestAudioProvider(t *testing.T) {
	builtins := map[string]Provider{
		"kg": &Kugou{}, "kw": &Kuwo{}, "wy": &Netease{}, "tx": &Tencent{},
	}
	withRegistry(map[string]Provider{}, func() {
		// No user source: the named provider answers.
		if got := AudioProvider(builtins["kw"], &Song{Source: "kw"}); got != builtins["kw"] {
			t.Errorf("AudioProvider = %v, want the built-in kw", idOrNil(got))
		}
	})

	merged := map[string]Provider{}
	for k, v := range builtins {
		merged[k] = v
	}
	merged["hyw"] = us("hyw", "")
	withRegistry(merged, func() {
		// With one configured, it replaces the built-in.
		got := AudioProvider(merged["kw"], &Song{Source: "kw"})
		if got.ID() != "hyw" {
			t.Errorf("AudioProvider = %v, want hyw", idOrNil(got))
		}
		// Asking for the user source directly must not bounce it back to itself.
		if got := AudioProvider(merged["hyw"], &Song{Source: "kw"}); got != merged["hyw"] {
			t.Errorf("AudioProvider(hyw) = %v, want hyw unchanged", idOrNil(got))
		}
		// A song with no platform cannot be routed.
		if got := AudioProvider(merged["kw"], &Song{}); got != merged["kw"] {
			t.Errorf("AudioProvider with empty source = %v, want the built-in kw", idOrNil(got))
		}
		// A nil song is the same - never a panic, never a reroute.
		if got := AudioProvider(merged["kw"], nil); got != merged["kw"] {
			t.Errorf("AudioProvider(nil) = %v, want the built-in kw", idOrNil(got))
		}
	})
}

// TestAudioProviderSourceWins pins the pinned-over-general rule end to end.
func TestAudioProviderSourceWins(t *testing.T) {
	merged := map[string]Provider{
		"kw":     &Kuwo{},
		"wy":     &Netease{},
		"hyw":    us("hyw", ""),
		"kwonly": us("kwonly", "kw"),
	}
	withRegistry(merged, func() {
		if got := AudioProvider(merged["kw"], &Song{Source: "kw"}); got.ID() != "kwonly" {
			t.Errorf("kw audio = %v, want the kw-pinned source", got.ID())
		}
		if got := AudioProvider(merged["wy"], &Song{Source: "wy"}); got.ID() != "hyw" {
			t.Errorf("wy audio = %v, want the general source", got.ID())
		}
	})
}

func idOrNil(p Provider) string {
	if p == nil {
		return "nil"
	}
	return p.ID()
}
