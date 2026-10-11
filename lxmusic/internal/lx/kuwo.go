package lx

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

/*
Kuwo, source id "kw".

Search, detail, cover and lyric all live in dist/renderer.js (the provider
object that starts with `regExps: { wordLine, tagLine, wordTimeAll, ... }`).
Resolving the audio URL is handed to the external User API plugin
(`action: "musicUrl"`), so MusicURL here uses kuwo's public anti-CDN gateway
instead.

  search  GET http://search.kuwo.cn/r.s?client=kt&all=...
  detail  GET http://www.kuwo.cn/api/www/music/musicInfo?mid=
  lyric   GET http://mlyric.kuwo.cn/mobi.s?f=web&type=lyric&rid=
  pic     GET http://artistpicserver.kuwo.cn/pic.web?type=rid_pic&rid=
  url     GET http://antiserver.kuwo.cn/anti.s?type=convert_url

The lyric transport is layered: "TP=content\r\nlrcx=1\r\n\r\n" + a zlib stream
of *base64 text*, which decodes to bytes that are XORed with the key "yeelion"
before they become the LRC text.
*/

const (
	kuwoSearch  = "http://search.kuwo.cn/r.s"
	kuwoDetail  = "http://www.kuwo.cn/api/www/music/musicInfo"
	kuwoLyric   = "http://mlyric.kuwo.cn/mobi.s"
	kuwoPic     = "http://artistpicserver.kuwo.cn/pic.web"
	kuwoConvert = "http://antiserver.kuwo.cn/anti.s"
	kuwoPage    = "http://www.kuwo.cn/play_detail/"
	kuwoUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) kwplayer_ar_9.2.2.1"
)

// kuwoLyricKey is the XOR key used by the app's decompression helper.
var kuwoLyricKey = []byte("yeelion")

// kuwoMInfo matches one rendition inside N_MINFO: level,bitrate,format,size.
var kuwoMInfo = regexp.MustCompile(`level:([\w-]+),bitrate:(\d+),format:([\w-]+),size:([\w.]+)`)

// kuwoTagLine matches the [ver:..] / [ti:..] metadata lines of the lyric. The
// app's own tagLine also matches [kuwo:..], but that line only carries the
// timing offset used to rebuild the per-word timings, so it is dropped here
// instead of being printed as lyric metadata.
var kuwoTagLine = regexp.MustCompile(`\[(?:ml|ver|ti|ar|al|offset|by|length|tool|sign):\s*[^]]*\]`)

// kuwoWordTime is rxps.wordTimeAll: the "<start,dur>" marker kuwo puts in front
// of every word. The app deletes all of them before the lyric is displayed.
var kuwoWordTime = regexp.MustCompile(`<-?\d+,-?\d+(?:,-?\d+)?>`)

// kuwoLine is one lyric line: the original lyric and, when lrcx=1, the Chinese
// translation that kuwo sends at the same timestamp.
type kuwoLine struct {
	time     string
	original string
	trans    string
}

// Kuwo implements Provider for kuwo music.
type Kuwo struct{ C *Client }

func (k *Kuwo) ID() string   { return "kw" }
func (k *Kuwo) Name() string { return "酷我音乐" }

// Anonymous audio: it is a short preview clip rather than the track, but it is
// a real file that lands on disk, which is what makes a row pickable. The CLI
// labels the clip instead of passing it off as the track.
func (k *Kuwo) AnonymousAudio() bool { return true }

// bitrateToType is the desktop app's exact switch.
func kuwoBitrateToType(b int64) string {
	switch b {
	case 4000:
		return QFlac24
	case 2000:
		return QFlac
	case 320:
		return Q320
	case 128:
		return Q128
	default:
		return ""
	}
}

// parseHumanSize turns kuwo's "52.83Mb" / "4.12Mb" sizes into bytes.
//
// The unit is two letters ("Mb" / "Kb") in the live response, so splitting on a
// single trailing character is wrong: it would leave "52.83Mb" for ParseFloat
// and every size would come back as 0, which is exactly what happened before
// the search table needed the value. The number is read up to the first letter,
// then the unit decides the multiplier. "Mb" must be tested before "Kb", since
// "mb" contains neither "k" nor a bare "m" boundary worth worrying about.
func parseHumanSize(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	i := 0
	for i < len(s) && (s[i] == '.' || s[i] >= '0' && s[i] <= '9') {
		i++
	}
	if i == 0 {
		return 0
	}
	mult := int64(1)
	switch {
	case strings.Contains(s[i:], "m"):
		mult = 1024 * 1024
	case strings.Contains(s[i:], "k"):
		mult = 1024
	}
	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0
	}
	return int64(n * float64(mult))
}

func (k *Kuwo) headers() map[string]string {
	return map[string]string{"User-Agent": kuwoUA, "Referer": "http://www.kuwo.cn/"}
}

// Search calls search.kuwo.cn.
func (k *Kuwo) Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	if page <= 0 {
		page = 1
	}
	u := fmt.Sprintf("%s?client=kt&all=%s&pn=%d&rn=%d&uid=794762570&ver=kwplayer_ar_9.2.2.1&vipver=1&show_copyright_off=1&newver=1&ft=music&cluster=0&strategy=2012&encoding=utf8&rformat=json&vermerge=1&mobi=1&issubtitle=1",
		kuwoSearch, url.QueryEscape(keyword), page-1, limit)

	var resp struct {
		Total   FInt `json:"TOTAL"`
		Show    FInt `json:"SHOW"`
		Abslist []struct {
			MusicRID FStr `json:"MUSICRID"`
			NMInfo   FStr `json:"N_MINFO"`
			SongName FStr `json:"SONGNAME"`
			Artist   FStr `json:"ARTIST"`
			AlbumID  FStr `json:"ALBUMID"`
			Album    FStr `json:"ALBUM"`
			Duration FStr `json:"DURATION"`
		} `json:"abslist"`
	}
	if err := k.C.GetJSON(ctx, u, k.headers(), &resp); err != nil {
		return nil, err
	}
	total := int(resp.Total)
	out := &SearchResult{
		Source: k.ID(), List: make([]*Song, 0, len(resp.Abslist)),
		Total: total, Page: page, Limit: limit,
		AllPage: (total + limit - 1) / limit,
	}
	seen := map[string]bool{}
	for _, it := range resp.Abslist {
		s := k.toSong(string(it.MusicRID), string(it.NMInfo), string(it.SongName),
			string(it.Artist), string(it.AlbumID), string(it.Album), string(it.Duration))
		if s == nil || seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		out.List = append(out.List, s)
	}
	return out, nil
}

// toSong mirrors handleResult(): the id is MUSICRID minus the "MUSIC_" prefix.
func (k *Kuwo) toSong(mid, nmInfo, name, artist, albumID, album, duration string) *Song {
	mid = strings.TrimPrefix(mid, "MUSIC_")
	if mid == "" {
		return nil
	}
	// Song titles come back HTML-escaped (&amp; and friends).
	name = html.UnescapeString(name)
	artist = html.UnescapeString(artist)
	if name == "" {
		return nil
	}

	var qualities []Quality
	for _, m := range kuwoMInfo.FindAllStringSubmatch(nmInfo, -1) {
		bitrate, _ := strconv.ParseInt(m[2], 10, 64)
		t := kuwoBitrateToType(bitrate)
		if t == "" {
			continue
		}
		if !contains(qualityTypes(qualities), t) {
			qualities = append(qualities, Quality{Type: t, Size: parseHumanSize(m[4])})
		}
	}
	if len(qualities) == 0 {
		qualities = []Quality{{Type: Q128}}
	}

	dur := 0
	if n, err := strconv.Atoi(duration); err == nil {
		dur = n
	}
	return &Song{
		Source:    k.ID(),
		ID:        mid,
		Name:      name,
		Singer:    artist,
		Album:     html.UnescapeString(album),
		AlbumID:   html.UnescapeString(albumID),
		Duration:  dur,
		Qualities: qualities,
		MID:       mid,
		Pic:       kuwoCoverURL(mid),
		PageURL:   kuwoPage + mid,
	}
}

func qualityTypes(qs []Quality) []string {
	out := make([]string, len(qs))
	for i, q := range qs {
		out[i] = q.Type
	}
	return out
}

func kuwoCoverURL(mid string) string {
	return fmt.Sprintf("%s?corp=kuwo&type=rid_pic&pictype=500&size=500&rid=%s", kuwoPic, mid)
}

// Detail refetches the song from the www API to fill in any missing metadata.
// That API answers code=0 to an anonymous client that has no login cookie, so
// when it does the title is recovered from the lyric's own [ti:] and [ar:]
// tags, which are public.
func (k *Kuwo) Detail(ctx context.Context, s *Song) error {
	mid := s.MID
	if mid == "" {
		mid = s.ID
	}
	var resp struct {
		Code FInt `json:"code"`
		Data struct {
			Name   FStr `json:"SONGNAME"`
			Artist FStr `json:"ARTIST"`
			Album  FStr `json:"ALBUM"`
		} `json:"data"`
	}
	err := k.C.GetJSON(ctx, kuwoDetail+"?mid="+url.QueryEscape(mid), k.headers(), &resp)
	if err == nil && resp.Code == 200 {
		if s.Name == "" {
			s.Name = html.UnescapeString(string(resp.Data.Name))
		}
		if s.Singer == "" {
			s.Singer = html.UnescapeString(string(resp.Data.Artist))
		}
		if s.Album == "" {
			s.Album = html.UnescapeString(string(resp.Data.Album))
		}
	}
	if s.Name == "" {
		if ly, lerr := k.Lyric(ctx, s); lerr == nil {
			s.Name = lyricTag(ly.Plain, "ti")
			if s.Singer == "" {
				s.Singer = lyricTag(ly.Plain, "ar")
			}
			if s.Album == "" {
				s.Album = lyricTag(ly.Plain, "al")
			}
		}
	}
	if s.Name == "" {
		if err != nil {
			return err
		}
		return fmt.Errorf("kw: could not resolve the title of %s", s.KeyString())
	}
	return nil
}

// lyricTag reads one "[ti:...]" style metadata tag out of an lrc body.
func lyricTag(lrc, tag string) string {
	for _, line := range strings.Split(lrc, "\n") {
		line = strings.TrimSpace(line)
		prefix := "[" + tag + ":"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		v, _, ok := strings.Cut(line, "]")
		if !ok {
			continue
		}
		return html.UnescapeString(strings.TrimSpace(v[len(prefix):]))
	}
	return ""
}

// Pic returns the 500x500 cover.
func (k *Kuwo) Pic(ctx context.Context, s *Song) (string, error) {
	if s.Pic != "" {
		return s.Pic, nil
	}
	s.Pic = kuwoCoverURL(s.ID)
	return s.Pic, nil
}

// Lyric fetches the synced lyric; lrcx=1 also asks for the translation.
func (k *Kuwo) Lyric(ctx context.Context, s *Song) (*Lyric, error) {
	u := fmt.Sprintf("%s?f=web&type=lyric&lrcx=1&rid=%s&encode=utf8", kuwoLyric, url.QueryEscape(s.ID))
	body, err := k.C.GetBytes(ctx, u, k.headers())
	if err != nil {
		return nil, err
	}
	text := KuwoDecompress(body)
	if text == "" {
		return nil, fmt.Errorf("kw: lyric not found for %s", s.KeyString())
	}
	ly := parseKuwoLRC(text)
	if ly == nil {
		return nil, fmt.Errorf("kw: lyric not found for %s", s.KeyString())
	}
	ly.URL = u
	return ly, nil
}

// KuwoDecompress runs the app's layered lyric decoder: strip the two header
// lines, inflate, base64-decode and XOR with "yeelion".
func KuwoDecompress(raw []byte) string {
	if !strings.HasPrefix(strings.ToLower(string(raw[:min(len(raw), 10)])), "tp=content") {
		return ""
	}
	if i := bytesIndexCRLF(raw); i > 0 {
		raw = raw[i:]
	}
	z, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	defer z.Close()
	buf, err := io.ReadAll(z)
	if err != nil {
		return ""
	}
	s := string(buf)
	if strings.TrimSpace(s) == "" {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return ""
	}
	out := append([]byte(nil), b...)
	for i := range out {
		out[i] ^= kuwoLyricKey[i%len(kuwoLyricKey)]
	}
	return string(out)
}

func bytesIndexCRLF(raw []byte) int {
	for i := 0; i+3 < len(raw); i++ {
		if raw[i] == '\r' && raw[i+1] == '\n' && raw[i+2] == '\r' && raw[i+3] == '\n' {
			return i + 4
		}
	}
	return 0
}

// parseKuwoLRC mirrors the app's parseLrc: tags at the top, timed lines at the
// bottom. The app's sortLrcArr pairs a repeated timestamp with the previous
// line, which both shifts kuwo's translation by one line and rewrites its time
// to the previous line's; kuwo actually sends the pair in the order translation
// then original at the *same* timestamp, so lines are grouped by time here.
//
// Kuwo prefixes every word with a "<start,dur>" marker. Lines whose markers are
// all "<0,0>" are the Chinese translation: they carry no per-word timing, which
// is what separates them from the original. With lrcx=0 there are no markers
// and no translation, so every line is an original.
func parseKuwoLRC(text string) *Lyric {
	var tags []string
	var lines []*kuwoLine
	pos := map[string]int{}

	src := strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")
	if src == "" {
		return nil
	}
	for _, raw := range strings.Split(src, "\n") {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		tag, rest, ok := stripFirstTimeTag(s)
		if !ok {
			if m := kuwoTagLine.FindString(s); m != "" {
				tags = append(tags, html.UnescapeString(m))
			}
			continue
		}
		ts := tag[1 : len(tag)-1]
		if strings.HasSuffix(ts, ".") {
			ts += "0"
		}
		rest = html.UnescapeString(strings.TrimSpace(rest))
		isTrans := strings.HasPrefix(rest, "<0,0>")
		line := strings.TrimSpace(kuwoWordTime.ReplaceAllString(rest, ""))
		i, seen := pos[ts]
		if !seen {
			lines = append(lines, &kuwoLine{time: ts})
			i = len(lines) - 1
			pos[ts] = i
		}
		l := lines[i]
		if isTrans {
			l.trans = line
		} else {
			l.original = line
		}
	}
	if len(lines) == 0 {
		return nil
	}
	out := &Lyric{Format: "lrc"}
	out.Plain = renderKuwoLRC(tags, lines, func(l *kuwoLine) string { return l.original })
	out.Raw = out.Plain
	out.Trans = renderKuwoLRC(tags, lines, func(l *kuwoLine) string { return l.trans })
	return out
}

// renderKuwoLRC prints the tag block then the selected field of every line,
// skipping the entries kuwo left empty.
func renderKuwoLRC(tags []string, lines []*kuwoLine, pick func(*kuwoLine) string) string {
	var b strings.Builder
	for _, tag := range tags {
		b.WriteString(tag)
		b.WriteString("\n")
	}
	for _, l := range lines {
		if text := strings.TrimSpace(pick(l)); text != "" {
			b.WriteString("[" + l.time + "]" + text + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// stripFirstTimeTag pulls the "[mm:ss.xx]" timestamp off the front of a line.
func stripFirstTimeTag(s string) (string, string, bool) {
	if len(s) < 2 || s[0] != '[' {
		return "", "", false
	}
	e := strings.IndexByte(s, ']')
	if e < 1 {
		return "", "", false
	}
	in := s[1:e]
	if !strings.Contains(in, ":") {
		return "", "", false
	}
	if !allDigitsDot(in) {
		return "", "", false
	}
	return s[:e+1], s[e+1:], true
}

func allDigitsDot(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || c == '.' || c == ':') {
			return false
		}
	}
	return true
}

// MusicURL asks the anti-CDN gateway for a converted URL.
//
// Two things the desktop app papers over with its User API plugins:
//
//   - The gateway replies with the bare URL and content-type text/html. There
//     is no JSON envelope, so decoding it as {"url":"..."} fails.
//   - Anonymous access gets mp3 only. format=flac and format=ape answer
//     "refuse request!", and whatever bitrate is asked for every song maps to
//     the same ~181 KB preview object - about 9 seconds of audio, not the
//     track. So lossless is tried and then walked down to 320k instead of
//     failing, which is what "best quality available" means for kuwo.
func (k *Kuwo) MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error) {
	q := quality.Type
	if q == "" || q == QAuto || q == QBestKnown {
		q = s.BestQuality()
	}
	want := q
	type offer struct {
		format string
		br     int64
		label  string
	}
	offers := []offer{{"mp3", 320, Q320}, {"mp3", 128, Q128}}
	if q == QFlac || q == QFlac24 {
		offers = append([]offer{{"flac", 2000, q}}, offers...)
	}
	var last string
	for _, o := range offers {
		u := fmt.Sprintf("%s?type=convert_url&rid=%s&format=%s&br=%d000", kuwoConvert, s.ID, o.format, o.br)
		body, err := k.C.GetBytes(ctx, u, k.headers())
		if err != nil {
			return nil, err
		}
		stream := strings.TrimSpace(string(body))
		if !strings.HasPrefix(stream, "http") {
			last = shortText(stream)
			continue
		}
		note := "kuwo serves one fixed short preview anonymously, so this is a clip rather than the track"
		if want != "" && want != o.label {
			note = "kuwo refused " + want + " anonymously; " + o.label + " was served instead"
		}
		return &PlayerURL{
			URL: stream, Quality: o.label, Source: k.ID(), Duration: s.Duration,
			FileName: s.Name + ".mp3", Note: note,
			Headers: map[string]string{"Referer": "http://www.kuwo.cn/"},
		}, nil
	}
	return nil, fmt.Errorf("kw: no url for %s: %q", s.KeyString(), last)
}

// shortText keeps a non-url gateway reply on one line for an error message.
func shortText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
