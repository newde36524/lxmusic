package lx

import "testing"

// TestFilterStr pins the punctuation set that stops counting as identity. A row
// named "晴天 (Live)" and one named "晴天(LIVE)" are the same song after this,
// which is what lets the relevance score match across sources.
func TestFilterStr(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"晴天", "晴天"},
		{"晴天 周杰伦", "晴天周杰伦"},
		{"晴天(现场版)", "晴天现场版"},
		{"晴天（现场版）", "晴天现场版"},
		{"A、B、C", "abc"},
		{"it's a test, ok", "itsatestok"},
		{"a<b>c|d/e[f]g!h", "abcdefgh"},
		{"", ""},
	} {
		if got := filterStr(c.in); got != c.want {
			t.Errorf("filterStr(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNormSingers checks that artist order does not affect identity, which is
// the case the desktop app hit most: the same song listed as "A、B" on one
// source and "B、A" on another.
func TestNormSingers(t *testing.T) {
	if a, b := normSingers("周杰伦、林俊杰"), normSingers("林俊杰、周杰伦"); a != b {
		t.Errorf("normSingers is not order insensitive: %q vs %q", a, b)
	}
	if got := normSingers("A|B&C;D"); got != "A、B、C、D" {
		t.Errorf("normSingers(%q) = %q, want %q", "A|B&C;D", got, "A、B、C、D")
	}
	if got := normSingers("独唱"); got != "独唱" {
		t.Errorf("normSingers left a single artist alone: %q", got)
	}
}

// TestRelevanceScore is the whole point of the ranking: a cover recorded in
// lossless must not outrank the original that is only available at 320k. The
// rows below are what a real 晴天 search looks like.
func TestRelevanceScore(t *testing.T) {
	for _, c := range []struct {
		name, singer, kw string
		want             int
	}{
		// Exact title.
		{"晴天", "周杰伦", "晴天", 3},
		{"晴天(吉他版)", "周杰伦", "晴天", 1}, // version marker, not an exact title
		// Every token covered.
		{"晴天", "周杰伦", "晴天 周杰伦", 3},
		{"晴天", "周杰伦、林俊杰", "晴天 周杰伦", 3},
		// Singer order does not matter.
		{"晴天", "林俊杰、周杰伦", "晴天 周杰伦", 3},
		// Right title, wrong artist: title matches but the artist token fails.
		{"晴天", "林俊杰", "晴天 周杰伦", 1},
		// Cover with the same title.
		{"晴天", "某某", "晴天 周杰伦", 1},
		// Whole keyword contained in the title.
		{"晴天", "周杰伦", "晴天", 3},
		// Keyword longer than the title.
		{"晴天", "周杰伦", "周杰伦晴天", 2},
		// A single token shows up somewhere.
		{"晴天", "周杰伦", "完全不相关的歌 周杰伦", 1},
		// Nothing at all.
		{"晴天", "周杰伦", "七里香 陈奕迅", 0},
		// Empty keyword is not a relation.
		{"晴天", "周杰伦", "", 0},
		{"晴天", "周杰伦", "   ", 0},
	} {
		if got := RelevanceScore(c.name, c.singer, c.kw); got != c.want {
			t.Errorf("RelevanceScore(%q, %q, %q) = %d, want %d", c.name, c.singer, c.kw, got, c.want)
		}
	}
}

// TestVersionAgree checks the tiebreaker: two rows that both match a keyword
// still differ if one is an alternate recording.
func TestVersionAgree(t *testing.T) {
	for _, c := range []struct {
		name, kw string
		want     bool
	}{
		{"晴天", "晴天", true},
		{"晴天(现场版)", "晴天", false},    // row has live, keyword does not
		{"晴天(现场版)", "晴天 现场", true},  // both have live
		{"晴天", "晴天 现场", false},      // keyword has live, row does not
		{"晴天 (Inst.)", "晴天", false}, // inst is a substring match
		{"晴天 伴奏版", "晴天", false},
		{"晴天", "晴天", true},
		{"晴天 (Cover)", "晴天", false},
	} {
		if got := VersionAgree(c.name, c.kw); got != c.want {
			t.Errorf("VersionAgree(%q, %q) = %v, want %v", c.name, c.kw, got, c.want)
		}
	}
}

// TestQueryTokens covers the separator set and the empty cases.
func TestQueryTokens(t *testing.T) {
	if toks := queryTokens("晴天 周杰伦"); len(toks) != 2 || toks[0] != "晴天" || toks[1] != "周杰伦" {
		t.Errorf("queryTokens split a plain keyword wrong: %v", toks)
	}
	if toks := queryTokens("晴天、周杰伦/林俊杰"); len(toks) != 3 {
		t.Errorf("queryTokens did not split every separator: %v", toks)
	}
	if toks := queryTokens(""); len(toks) != 0 {
		t.Errorf("queryTokens(\"\") = %v, want empty", toks)
	}
	if toks := queryTokens("晴天"); len(toks) != 1 || toks[0] != "晴天" {
		t.Errorf("queryTokens on one word: %v", toks)
	}
}
