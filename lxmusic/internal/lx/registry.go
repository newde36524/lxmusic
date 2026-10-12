package lx

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrNoSearch is what a source returns instead of an error when it cannot
// search by design. It is a sentinel, not a failure: a sweep across every
// source should skip it in silence, while a user who asks for it by name still
// deserves the explanation.
var ErrNoSearch = errors.New("does not search")

// Provider is the contract every platform adapter fulfils. It is shaped after
// the per-source object graph the desktop app builds in renderer.js, which is
// also where the three User API actions ("musicUrl", "lyric", "pic") land.
type Provider interface {
	ID() string
	Name() string
	// Search looks up songs by keyword. A source that does not search by design
	// wraps ErrNoSearch in its error, so a caller can tell "not applicable" from
	// "the request failed" - see ErrNoSearch.
	Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error)
	MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error)
	Lyric(ctx context.Context, s *Song) (*Lyric, error)
	Pic(ctx context.Context, s *Song) (string, error)
	Detail(ctx context.Context, s *Song) error
	// AnonymousAudio reports whether MusicURL can hand over a playable audio
	// url with no login and no user source in between. The search table lists
	// only rows a user can actually download, so this decides whether a source
	// contributes rows to it at all. kw answers true - it serves a short
	// preview clip, which is still a file - and mg answers true too, from the
	// gateway's listen-url route, which hands over a 128k MP3 to an
	// unauthenticated client even though the desktop app ships no mg audio
	// implementation. kg, wy and tx answer false: an audio url is precisely
	// the capability v2.12.6 outsourced to the user source scripts, and
	// guessing it here is what made a pickable row that could never download.
	AnonymousAudio() bool
}

var registry = map[string]Provider{}

// Register binds one shared Client to every platform provider and to each user
// source listed in the config. User sources are added on top of the built-ins,
// so a config entry named "kg" would shadow the built-in kugou adapter — the
// config is the user's own, and that is the point of the feature.
func Register(c *Client) {
	registry = map[string]Provider{
		"kg": &Kugou{C: c},
		"kw": &Kuwo{C: c},
		"wy": &Netease{C: c},
		"tx": &Tencent{C: c},
		"mg": &Migu{C: c},
	}
	if cfg, err := LoadConfig(); err == nil {
		for _, spec := range cfg.Sources {
			if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.URL) == "" {
				continue
			}
			registry[strings.ToLower(spec.Name)] = &UserSource{C: c, Spec: spec}
		}
	}
}

// sourcesOrder is the display order the desktop app uses in its source picker.
var sourcesOrder = []string{"kg", "kw", "wy", "tx", "mg"}

// Sources returns the registered source ids in display order.
func Sources() []string {
	out := make([]string, 0, len(sourcesOrder))
	for _, id := range sourcesOrder {
		if _, ok := registry[id]; ok {
			out = append(out, id)
		}
	}
	extra := make([]string, 0)
	for id := range registry {
		if !contains(out, id) {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// GetProvider looks up one adapter by source id.
func GetProvider(id string) (Provider, error) {
	p, ok := registry[strings.ToLower(id)]
	if !ok {
		return nil, fmt.Errorf("unknown source %q (known: %s)", id, strings.Join(Sources(), ", "))
	}
	return p, nil
}

// UserSourceFor returns the user source that should serve audio for a platform,
// or nil when none is configured. A source pinned to the platform (--source kw)
// beats one left general (--source "" = any platform), since the pinned one was
// written for it. Only Spec.Source is consulted, never the script's own
// declared capabilities - those need a node round trip to read, and the script
// is exactly the thing being asked for the url.
func UserSourceFor(platform string) Provider {
	want := strings.ToLower(strings.TrimSpace(platform))
	if want == "" {
		// Nothing to route on: a song with no platform was parsed badly, and a
		// general source cannot tell which platform to hand it to either.
		return nil
	}
	var general Provider
	for _, id := range Sources() {
		u, ok := registry[id].(*UserSource)
		if !ok {
			continue
		}
		src := strings.ToLower(strings.TrimSpace(u.Spec.Source))
		if src != "" && src == want {
			return u
		}
		if src == "" && general == nil {
			general = u
		}
	}
	return general
}

// AudioProvider picks who serves audio for s. A configured user source wins: a
// platform's anonymous channel is precisely the capability the desktop app hands
// out to user sources, so with one configured the built-in answer is the preview
// clip, not the track. p stays the fallback - an unreachable script should not
// make a song undownloadable. A user source the user named outright is left
// alone, since it already is the user source.
func AudioProvider(p Provider, s *Song) Provider {
	if s == nil {
		return p
	}
	if u := UserSourceFor(s.Source); u != nil && u != p {
		return u
	}
	return p
}

// Hydrate fills the metadata a bare "--key source:id" cannot carry on its own.
// It asks the adapter the song belongs to, not the one the user selected: with
// "--source flower --key kw:6686351" the song is a kuwo song and flower cannot
// tell us its title, so kuwo answers. Without this a user source receives an
// empty song name and most published scripts refuse the request.
func Hydrate(ctx context.Context, p Provider, s *Song) error {
	if s == nil || s.Name != "" {
		return nil
	}
	q := p
	if s.Source != "" && s.Source != p.ID() {
		if r, err := GetProvider(s.Source); err == nil && r != nil {
			q = r
		}
	}
	return q.Detail(ctx, s)
}

// SongFromKey turns a "source:id" reference into a Song. The id is enough for
// URL/pic resolution on every source; kugou's id additionally carries the
// 128k hash, and lyric resolution additionally needs a song name.
func SongFromKey(p Provider, key string) (*Song, error) {
	src, id, ok := strings.Cut(key, ":")
	if !ok {
		// no source prefix: assume this provider
		src, id = p.ID(), key
	}
	if src == "" || id == "" {
		return nil, fmt.Errorf("bad song key %q (want source:id, e.g. kg:20505418B3A5...)", key)
	}
	s := &Song{Source: src, ID: id}
	switch src {
	case "kg":
		if len(id) >= 32 {
			s.Hash = id[len(id)-32:]
			s.MID = id[:len(id)-32]
		}
	case "kw", "wy", "tx":
		s.MID = id
	}
	return s, nil
}

// SongFromSearchResult picks song n out of a search, which is the reliable way
// to get a fully hydrated song (name, duration, quality matrix).
func SongFromSearchResult(res *SearchResult, index int) (*Song, error) {
	if res == nil || len(res.List) == 0 {
		return nil, fmt.Errorf("empty search result")
	}
	if index < 1 || index > len(res.List) {
		return nil, fmt.Errorf("index %d out of range (1-%d)", index, len(res.List))
	}
	return res.List[index-1], nil
}

// SongRef resolves the CLI's "--key" / "--search" inputs into a Song.
func SongRef(ctx context.Context, p Provider, key, search string, index, page, limit int) (*Song, error) {
	if search != "" {
		res, err := p.Search(ctx, search, page, limit)
		if err != nil {
			return nil, err
		}
		return SongFromSearchResult(res, index)
	}
	if key != "" {
		return SongFromKey(p, key)
	}
	return nil, fmt.Errorf("pass --key source:id or --search keyword")
}

// SearchAll queries every source and returns them in display order.
func SearchAll(ctx context.Context, keyword string, page, limit int) ([]*SearchResult, error) {
	var out []*SearchResult
	var firstErr error
	for _, id := range Sources() {
		p := registry[id]
		res, err := p.Search(ctx, keyword, page, limit)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %v", id, err)
			}
			continue
		}
		out = append(out, res)
	}
	return out, firstErr
}
