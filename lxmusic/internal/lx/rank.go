package lx

import "strings"

/*
Search ranking.

The desktop app never shows a raw keyword search: it normalizes every hit and
matches it against the identity it is looking for, with a hard check that the
version markers agree. This file ports that matching as a *relevance score*,
which is what a keyword search needs — there is no target song to match, only a
query to measure proximity to.

The score is the primary sort key the table uses, ahead of quality and size.
Without it a cover recorded in lossless outranks the original that is only
available at 320k, which is the failure mode the README called out.
*/

// filterStr normalizes text for identity comparison: strip punctuation and
// whitespace, lowercase. Copied from findMusic's filterStr.
func filterStr(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ' || r == '\t':
		case r == '\'' || r == '.' || r == ',' || r == '，' || r == '&' || r == '"':
		case r == '、' || r == '(' || r == ')' || r == '（' || r == '）':
		case r == '`' || r == '~' || r == '<' || r == '>' || r == '|' || r == '/' || r == '!' || r == '！':
		case r == '[' || r == ']':
		default:
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

// normSingers reorders a multi-artist string so that "A、B" and "B、A" compare
// equal. findMusic's sortSingle, which splits on the singer separators, sorts the
// pieces and rejoins them with 、.
func normSingers(s string) string {
	pieces := strings.FieldsFunc(s, func(r rune) bool {
		return r == '、' || r == '&' || r == ';' || r == '；' || r == '/' || r == ',' || r == '，' || r == '|'
	})
	for i, p := range pieces {
		pieces[i] = strings.TrimSpace(p)
	}
	for i := 1; i < len(pieces); i++ {
		for j := i; j > 0 && pieces[j] < pieces[j-1]; j-- {
			pieces[j], pieces[j-1] = pieces[j-1], pieces[j]
		}
	}
	return strings.Join(pieces, "、")
}

// queryTokens splits a free-text keyword into the pieces a hit has to cover.
// "晴天 周杰伦" -> ["晴天", "周杰伦"]; a single word is one token.
func queryTokens(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '、' || r == '&' || r == ';' || r == '；' ||
			r == '/' || r == ',' || r == '，' || r == '|' || r == '(' || r == ')' ||
			r == '（' || r == '）'
	})
}

// RelevanceScore rates how closely one hit matches a keyword, best first:
//
//	3  the title is the keyword, exactly
//	2  every token of the keyword is covered by the title or the artists
//	1  the title contains the keyword, the keyword contains the title, or any
//	     single token shows up at all
//	0  no textual relation
//
// It is deliberately cheap and total: every row gets a number, so the sort never
// has to special-case a row the scorer did not understand.
//
// A hit whose title carries a version the keyword does not — "晴天(吉他版)" for
// the query "晴天" — is capped at 1. Without the cap it outranks the original on
// tokens alone, because "晴天气TAB版" still contains "晴天" and that is score 2.
func RelevanceScore(name, singer, keyword string) int {
	q := filterStr(keyword)
	if q == "" {
		return 0
	}
	nm := filterStr(name)
	sg := filterStr(normSingers(singer))
	if nm == q {
		return 3
	}
	toks := queryTokens(keyword)
	total, covered := 0, 0
	titleWhole := false
	for _, tok := range toks {
		t := filterStr(tok)
		if t == "" {
			continue
		}
		total++
		if t == nm {
			titleWhole = true
		}
		if strings.Contains(nm, t) || strings.Contains(sg, t) {
			covered++
		}
	}
	if total == 0 {
		return 0
	}
	score := 0
	switch {
	// Full coverage counts as an exact title when the title itself is one of the
	// tokens: "晴天" for "晴天 周杰伦" is still the song the user asked for.
	case covered == total && titleWhole:
		score = 3
	// A keyword with no separator at all yields one long token that contains
	// neither the title nor the artists ("周杰伦晴天" vs "晴天"/"周杰伦"). When
	// the keyword contains both, the user is still asking for this song.
	case covered == total,
		nm != "" && sg != "" && strings.Contains(q, nm) && strings.Contains(q, sg):
		score = 2
	case strings.Contains(nm, q), strings.Contains(q, nm), covered > 0:
		score = 1
	}
	if score > 1 && !VersionAgree(name, keyword) {
		score = 1
	}
	return score
}

// VersionAgree reports whether a hit carries the same version markers as the
// keyword does. Both sides must contain the same markers for every entry of the
// list, which is findMusic's isEqualsVersionMusicNameChar turned into a
// tiebreaker: a hit for "晴天" and a hit for "晴天(现场版)" both match the
// keyword "晴天", but only the first one agrees on version. Exported so the
// table sort and the score can share the rule.
func VersionAgree(name, keyword string) bool {
	nm := strings.ToLower(name)
	q := strings.ToLower(keyword)
	for _, c := range versionChars {
		if strings.Contains(nm, c) != strings.Contains(q, c) {
			return false
		}
	}
	return true
}

// versionChars is the marker list findMusic compares against, one per kind of
// alternate recording. Matching is a substring test, which is why "inst" also
// catches "instrumental".
var versionChars = []string{
	// instrumental
	"instrumental", "inst", "karaoke", "backing track", "minus one", "minus vocal",
	"no vocal", "off vocal", "vocal removed",
	"伴奏", "伴唱", "卡拉ok", "纯音乐", "无人声", "去人声", "消音",
	"オフボーカル", "インスト", "반주", "노래방",
	// live
	"live", "concert", "tour", "festival", "stage", "session",
	"现场", "演唱会", "巡演", "音乐会", "ライブ", "공연", "라이브",
	// cover
	"cover", "tribute", "rendition", "remake",
	"翻唱", "致敬", "改编", "カバー", "커버",
	// remix
	"remix", "mix", "club", "dance", "extended", "house", "edm", "dj", "dub", "trap",
	"混音", "重混", "电音", "リミックス", "리믹스",
	// acoustic
	"acoustic", "unplugged", "stripped",
	"原声", "不插电", "アコースティック", "어쿠스틱",
	// demo
	"demo", "小样", "试听", "デモ", "데모",
	// speed
	"slowed", "sped up", "speed up", "nightcore",
	"慢速", "加速", "变速", "ナイトコア",
	// special instrument
	"piano", "guitar", "orchestra", "symphony", "violin",
	"钢琴", "吉他", "管弦乐", "小提琴", "ピアノ", "ギター", "피아노", "기타",
}
