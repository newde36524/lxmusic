// Package lx implements the music platform protocols reverse engineered from
// lx-music-desktop v2.12.6 (resources/app.asar -> dist/renderer.js).
package lx

import "strings"

// Quality identifiers, kept identical to the desktop app's own vocabulary so
// results read the same way as the original UI.
const (
	Q128       = "128k"
	Q320       = "320k"
	QFlac      = "flac"
	QFlac24    = "flac24bit"
	QAuto      = "auto" // pick the best quality that is available
	QBestKnown = "best"
)

// AudioExt picks a file extension for a stream. The URL's own path extension
// wins, because a source can hand back a different format than the label it was
// asked for and the path is what the server actually serves - kuwo writes the
// flac renditions under "trackmedia/F000000B...flac". It falls back to the
// quality label. Only the path is inspected, since kuwo's query string is
// "$"-delimited and carries ".apk" in its spoofed User-Agent.
func AudioExt(quality, url string) string {
	if i := strings.LastIndex(url, "/"); i >= 0 {
		p := url[i+1:]
		if j := strings.IndexAny(p, "?#"); j >= 0 {
			p = p[:j]
		}
		if dot := strings.LastIndexByte(p, '.'); dot > 0 {
			switch strings.ToLower(p[dot:]) {
			case ".flac", ".mp3", ".ogg", ".m4a", ".aac", ".wav":
				return strings.ToLower(p[dot:])
			}
		}
	}
	if quality == QFlac || quality == QFlac24 {
		return ".flac"
	}
	return ".mp3"
}

// Quality is one downloadable rendition of a song.
type Quality struct {
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"` // bytes, 0 = unknown
	Hash string `json:"hash,omitempty"` // kugou only: the hash that selects this rendition
}

// Song is the normalized record produced by every provider.
//
// The field names mirror lx-music's internal shape (songmid / albumId / hash /
// types ...) so that the JSON output of this CLI is comparable with the app's.
type Song struct {
	Source    string    `json:"source"` // kg | kw | wy | tx
	ID        string    `json:"id"`     // stable id used by url/lyric/pic
	Name      string    `json:"name"`
	Singer    string    `json:"singer"`
	Album     string    `json:"album,omitempty"`
	AlbumID   string    `json:"albumId,omitempty"`
	Duration  int       `json:"duration,omitempty"` // seconds
	Pic       string    `json:"pic,omitempty"`
	Qualities []Quality `json:"qualities,omitempty"`
	Hash      string    `json:"hash,omitempty"`   // kugou default (128k) hash
	MID       string    `json:"mid,omitempty"`    // kugou audioid / tx song mid
	NumericID string    `json:"songId,omitempty"` // tx: numeric id, required for lyrics
	PageURL   string    `json:"pageUrl,omitempty"`
	Raw       any       `json:"-"` // provider-specific payload, kept for debugging
}

// KeyString is the canonical "source:id" reference used by the CLI.
func (s *Song) KeyString() string { return s.Source + ":" + s.ID }

// BestQuality returns the highest available quality that is not marked as a
// paid/encrypted VIP variant.
func (s *Song) BestQuality() string {
	for _, order := range []string{QFlac24, QFlac, Q320, Q128} {
		if s.HasQuality(order) {
			return order
		}
	}
	return ""
}

// bestOf picks the highest of a set of quality names by the fidelity order the
// table sorts by, returning "" when none of them is recognised.
func bestOf(qualitys []string) string {
	for _, order := range []string{QFlac24, QFlac, Q320, Q128} {
		for _, q := range qualitys {
			if q == order {
				return q
			}
		}
	}
	return ""
}

func (s *Song) HasQuality(t string) bool {
	for _, q := range s.Qualities {
		if q.Type == t {
			return true
		}
	}
	return false
}

// BestQualitySize is the byte count the platform announced for the best
// rendition the song offers - the rendition "--quality auto" would pick - or 0
// when the platform did not announce one. Matching BestQuality exactly matters:
// QualityByType falls back to the first quality when the type is unknown, which
// would report a lower rendition's size.
func (s *Song) BestQualitySize() int64 {
	b := s.BestQuality()
	if b == "" {
		return 0
	}
	for _, q := range s.Qualities {
		if q.Type == b {
			return q.Size
		}
	}
	return 0
}

// QualityByType resolves a requested quality to a concrete rendition, applying
// the same fallback rules the desktop player uses.
func (s *Song) QualityByType(t string) Quality {
	if t == QAuto || t == QBestKnown || t == "" {
		if b := s.BestQuality(); b != "" {
			t = b
		}
	}
	for _, q := range s.Qualities {
		if q.Type == t {
			return q
		}
	}
	if len(s.Qualities) > 0 { // unknown quality requested: use the first known one
		return s.Qualities[0]
	}
	return Quality{Type: t}
}

// SearchResult is one page of a search.
type SearchResult struct {
	Source  string  `json:"source"`
	List    []*Song `json:"list"`
	Total   int     `json:"total,omitempty"`
	Page    int     `json:"page,omitempty"`
	Limit   int     `json:"limit,omitempty"`
	AllPage int     `json:"allPage,omitempty"`
}

// Lyric is a normalized lyric payload.
type Lyric struct {
	Plain  string `json:"lyric,omitempty"`   // LRC text (translated lyric stripped)
	Trans  string `json:"tlyric,omitempty"`  // translated LRC
	Roman  string `json:"rlyric,omitempty"`  // romanized LRC
	Raw    string `json:"rawlric,omitempty"` // original payload, kugou KRC still encoded
	Format string `json:"format,omitempty"`  // lrc | krc
	URL    string `json:"url,omitempty"`     // set when fetched through a direct URL
}

// PlayerURL is the result of resolving a song to a playable/downloadable URL.
type PlayerURL struct {
	URL      string            `json:"url"`
	Quality  string            `json:"quality"`
	Headers  map[string]string `json:"headers,omitempty"`
	Size     int64             `json:"size,omitempty"`
	Duration int               `json:"duration,omitempty"`
	FileName string            `json:"fileName,omitempty"`
	Source   string            `json:"source"`
	Note     string            `json:"note,omitempty"` // what the provider had to compromise on
}
