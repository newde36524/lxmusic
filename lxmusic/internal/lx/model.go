// Package lx implements the music platform protocols reverse engineered from
// lx-music-desktop v2.12.6 (resources/app.asar -> dist/renderer.js).
package lx

import "strings"

// Quality identifiers, kept identical to the desktop app's own vocabulary so
// results read the same way as the original UI. The set matches upstream
// QUALITYS (common/constants.ts): ['flac24bit','flac','wav','ape','320k','192k','128k'].
const (
	QFlac24    = "flac24bit"
	QFlac      = "flac"
	QWav       = "wav"
	QApe       = "ape"
	Q320       = "320k"
	Q192       = "192k"
	Q128       = "128k"
	QAuto      = "auto" // pick the best quality that is available
	QBestKnown = "best"
)

// qualityOrder is the fidelity ranking, best first, in the desktop app's own
// order. Every quality decision — auto selection, the search table's sort key,
// the quality column — reads from this one list, so a rendition the built-in
// adapters never produce (wav, ape, 192k) still ranks where it belongs when a
// user source announces it, instead of falling through as an unknown tier.
var qualityOrder = []string{QFlac24, QFlac, QWav, QApe, Q320, Q192, Q128}

// QualityRank returns a fidelity rank for one quality label, best first, or -1
// when the label is unknown. Exported so the table sort in main and the
// provider's own ranking cannot drift apart.
func QualityRank(q string) int {
	for i, name := range qualityOrder {
		if q == name {
			return len(qualityOrder) - i
		}
	}
	return -1
}

// QualityList is the vocabulary a user may pass to --quality, plus "auto".
func QualityList() []string {
	out := make([]string, 0, len(qualityOrder)+1)
	for _, q := range qualityOrder {
		out = append(out, q)
	}
	return append(out, QAuto)
}

// PathExt is the audio extension on the path of a URL, lowercased, or "" when
// the path carries none. Only the path is inspected: a query string can carry an
// extension of its own, kuwo's does ("$format$flac", plus ".apk" inside a
// spoofed User-Agent), and so would corrupt the answer.
func PathExt(url string) string {
	i := strings.LastIndex(url, "/")
	if i < 0 {
		return ""
	}
	p := url[i+1:]
	if j := strings.IndexAny(p, "?#"); j >= 0 {
		p = p[:j]
	}
	dot := strings.LastIndexByte(p, '.')
	if dot <= 0 {
		return ""
	}
	switch e := strings.ToLower(p[dot:]); e {
	case ".flac", ".mp3", ".ogg", ".m4a", ".aac", ".wav", ".ape":
		return e
	}
	return ""
}

// AudioExt picks a file extension for a stream. The URL's own path extension
// wins, because a source can hand back a different format than the label it was
// asked for and the path is what the server actually serves - kuwo writes the
// flac renditions under "trackmedia/F000000B...flac". It falls back to the
// quality label.
func AudioExt(quality, url string) string {
	if e := PathExt(url); e != "" {
		return e
	}
	switch quality {
	case QFlac, QFlac24:
		return ".flac"
	case QWav:
		return ".wav"
	case QApe:
		return ".ape"
	default:
		return ".mp3"
	}
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
// ID is always the id the platform's own endpoints take, which is what makes a
// printed key resolvable: for mg that is the song id, not the copyright id the
// desktop app shows.
type Song struct {
	Source    string    `json:"source"` // kg | kw | wy | tx | mg
	ID        string    `json:"id"`     // stable id used by url/lyric/pic
	Name      string    `json:"name"`
	Singer    string    `json:"singer"`
	Album     string    `json:"album,omitempty"`
	AlbumID   string    `json:"albumId,omitempty"`
	Duration  int       `json:"duration,omitempty"` // seconds
	Pic       string    `json:"pic,omitempty"`
	Qualities []Quality `json:"qualities,omitempty"`
	Hash      string    `json:"hash,omitempty"`   // kugou default (128k) hash
	MID       string    `json:"mid,omitempty"`    // second id: kugou audioid, tx song mid, mg copyrightId
	NumericID string    `json:"songId,omitempty"` // tx: numeric id, required for lyrics
	LrcURL    string    `json:"lrcUrl,omitempty"` // mg: plain lyric url
	MrcURL    string    `json:"mrcUrl,omitempty"` // mg: per-word lyric url, encrypted
	TrcURL    string    `json:"trcUrl,omitempty"` // mg: translated lyric url
	PageURL   string    `json:"pageUrl,omitempty"`
	Raw       any       `json:"-"` // provider-specific payload, kept for debugging
}

// KeyString is the canonical "source:id" reference used by the CLI.
func (s *Song) KeyString() string { return s.Source + ":" + s.ID }

// BestQuality returns the highest available quality that is not marked as a
// paid/encrypted VIP variant.
func (s *Song) BestQuality() string {
	for _, order := range qualityOrder {
		if s.HasQuality(order) {
			return order
		}
	}
	return ""
}

// bestOf picks the highest of a set of quality names by the fidelity order the
// table sorts by, returning "" when none of them is recognised.
func bestOf(qualitys []string) string {
	for _, order := range qualityOrder {
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

// QualitySize is the byte count the platform announced for one rendition, or 0
// when it announced no size for it - which the table prints as "-", never as a
// guess borrowed from a different tier.
func (s *Song) QualitySize(t string) int64 {
	for _, q := range s.Qualities {
		if q.Type == t {
			return q.Size
		}
	}
	return 0
}

// BestQualitySize is the byte count the platform announced for the best
// rendition the song offers - the rendition "--quality auto" would pick - or 0
// when the platform did not announce one. Routing through QualitySize keeps the
// tier exact: an approximate match would report a different rendition's number.
func (s *Song) BestQualitySize() int64 {
	return s.QualitySize(s.BestQuality())
}

// QualityByType resolves a requested quality to a concrete rendition, applying
// the same fallback rules the desktop player uses.
//
// A tier the matrix does not list is passed through unchanged, not replaced by
// the first known one. A matrix can be thinner than the gateway's real
// inventory: mg announces a song known only by id with PQ alone, so substituting
// turned an explicit request for flac24bit into 128k with no note at all. Each
// provider already knows what to do with a tier it was not offered, and mg in
// particular reports the compromise it had to make. A caller that wants the size
// of a rendition must use BestQualitySize, which does not route through here.
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
