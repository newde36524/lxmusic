package lx

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

/*
Migu, source id "mg".

Search is lifted from src/renderer/utils/musicSdk/mg/ of lx-music-desktop
2.12.6. Everything else is not, because the app's copy of mg is hollow:
mg.getMusicUrl calls apis('mg') and api-source-info's sources list is empty for
mg, so the app answers "Api is not found", and the only audio implementation left
in the tree is a commented-out third-party proxy in mg/api-test.js. Detail,
cover, the lyric urls and the stream url all come from the gateway's listen-url
route instead, which the app never calls.

  search   GET  https://jadeite.migu.cn/music_search/v3/search/searchAll   MD5-signed
  detail   GET  .../MIGUM2.0/v2.0/content/listen-url?netType=00&resourceType=2&songId=<songId>&toneFlag=<PQ|HQ|SQ|ZQ24>
           returns data.songItem (name, album, artists, length, albumImgs, lrcUrl,
           mrcUrl, trcUrl) and data.url (the rendition itself)
  pic      albumImgs from songItem - no request of its own
  audio    data.url - anonymous, needs no headers, bound to the client ip and
           expiring in hours (the Tim query param)
  lyric    GET  <mrcUrl|lrcUrl>  TEA-encrypted hex, UTF-16LE inside

Three things about listen-url are worth knowing before changing anything here.

  It takes the song id, not the copyright id. resourceType 2 or 3 with a songId
  answers; a copyrightId answers "查询歌曲资源为空" or "歌曲下线" and there is no
  endpoint that maps one onto the other - resourceinfo.do used to, and it now
  returns code 000000 with an empty resource array for every id variant tried.
  That is why Song.ID is the song id for mg, and copyrightId sits in MID where
  the public song page link is built from it.

  It is idempotent across tones but not permissive. HQ, SQ and ZQ24 answer code
  000000 with an empty url for every song probed, so anonymous entitlement stops
  at PQ. MusicURL walks the ladder down and reports the compromise in Note
  rather than failing, because code 000000 plus no url is not a refusal the
  client can act on without guessing.

  The search announces the song's full rendition matrix, which the gateway will
  not serve above PQ. The matrix is kept as announced - it is the song's real
  ceiling and the sort key depends on it - and the downgrade is surfaced at the
  point of resolution.
*/

const (
	mgSearchURL      = "https://jadeite.migu.cn/music_search/v3/search/searchAll"
	mgListenURL      = "https://app.c.nf.migu.cn/MIGUM2.0/v2.0/content/listen-url"
	mgCDN            = "http://d.musicapp.migu.cn"
	mgPage           = "http://music.migu.cn/v3/music/song/"
	mgOK             = "000000"
	mgNoResourceCode = "200000"
	mgDeviceID       = "963B7AA0D21511ED807EE5846EC87D20"
	mgSignSeed       = "6cdc72a439cef99a3418d2a78aa28c73"
	mgSignApp        = "yyapp2d16148780a1dcc7408e06336b98cfd50"
	mgChannel        = "0146921"
	mgUIVersion      = "A_music_3.6.1"
	mgSearchUA       = "Mozilla/5.0 (Linux; U; Android 11.0.0; zh-cn; MI 11 Build/OPR1.170623.032) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
	mgLyricUA        = "Mozilla/5.0 (Linux; Android 5.1.1; Nexus 6 Build/LYZ28E) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Mobile Safari/537.36"
	mgListenUA       = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36"
	mgSearchSwitch   = `%7B%22song%22%3A1%2C%22album%22%3A0%2C%22singer%22%3A0%2C%22tagSong%22%3A1%2C%22mvSong%22%3A0%2C%22bestShow%22%3A1%2C%22songlist%22%3A0%2C%22lyricSong%22%3A0%7D`
)

// errMiguNoResource is what listen-url answers when it recognises the id shape
// but has nothing to give. It covers two cases that cannot be told apart: a
// song that is genuinely offline, and a copyright id passed where a song id was
// wanted. Callers translate it with mgIDNote.
var errMiguNoResource = errors.New("listen-url has no resource for this id")

// mgIDNote explains why a bare mg reference failed. It is appended only when the
// gateway answered errMiguNoResource, never for a network error.
const mgIDNote = "mg keys are song ids; one copied from the desktop app is a copyright id. get a song id from: lxmusic search --source mg"

// Migu implements Provider for migu music.
type Migu struct{ C *Client }

func (m *Migu) ID() string   { return "mg" }
func (m *Migu) Name() string { return "咪咕音乐" }

// AnonymousAudio is true: listen-url hands over a 128k MP3 to an unauthenticated
// client, which is exactly the capability the search table gates on. The ceiling
// is PQ, not the matrix the search announces - see MusicURL.
func (m *Migu) AnonymousAudio() bool { return true }

// mgTone maps a quality label onto the toneFlag the gateway wants. The full
// vocabulary is LQ|PQ|HQ|SQ|ZQ|Z3D|ZQ24|ZQ32|AV3A; only these four line up with
// the app's quality labels. ZQ24 is the 24-bit tier, spelled ZQ in some replies.
var mgTone = map[string]string{
	Q128:    "PQ",
	Q320:    "HQ",
	QFlac:   "SQ",
	QFlac24: "ZQ24",
}

// mgToneLadder is the tones best-first. MusicURL walks it when a tier comes back
// with no url, stopping at the first one the gateway will serve.
var mgToneLadder = []string{"ZQ24", "SQ", "HQ", "PQ"}

// mgLadderFrom returns the tones from t downward, the range a request for t is
// allowed to fall through.
func mgLadderFrom(tone string) []string {
	for i, t := range mgToneLadder {
		if t == tone {
			return mgToneLadder[i:]
		}
	}
	return []string{mgTone[Q128]}
}

// mgQualityForTone turns a toneFlag back into a quality label.
func mgQualityForTone(tone string) string {
	if q := mgFormatTypes[tone]; q != "" {
		return q
	}
	return Q128
}

// ------------------------------------------------------------- search

type mgArtist struct {
	Name FStr `json:"name"`
}

// mgFormat is one entry of audioFormats (search) or newRateFormats
// (resourceinfo). The size field differs between the two endpoints, so all four
// spellings are read and the first non-zero one wins. Upstream uses "??" which
// would keep a size of 0; a zero size is a missing value, not a rendition.
type mgFormat struct {
	FormatType  FStr `json:"formatType"`
	Asize       FInt `json:"asize"`
	Isize       FInt `json:"isize"`
	Size        FInt `json:"size"`
	AndroidSize FInt `json:"androidSize"`
}

func (f mgFormat) SizeBytes() int64 {
	for _, v := range []int64{int64(f.Asize), int64(f.Isize), int64(f.Size), int64(f.AndroidSize)} {
		if v != 0 {
			return v
		}
	}
	return 0
}

// mgSearchItem is one song of songResultData.resultList, which is an array of
// arrays - every group has to be walked. mrcurl is lowercase here.
type mgSearchItem struct {
	SongID       FStr       `json:"songId"`
	CopyrightID  FStr       `json:"copyrightId"`
	Name         FStr       `json:"name"`
	Album        FStr       `json:"album"`
	AlbumID      FStr       `json:"albumId"`
	ArtistName   FStr       `json:"artistName"`
	Duration     FInt       `json:"duration"`
	SingerList   []mgArtist `json:"singerList"`
	AudioFormats []mgFormat `json:"audioFormats"`
	Img1         FStr       `json:"img1"`
	Img2         FStr       `json:"img2"`
	Img3         FStr       `json:"img3"`
	LrcURL       FStr       `json:"lrcUrl"`
	MrcURL       FStr       `json:"mrcUrl"`
	MrcURLSnake  FStr       `json:"mrcurl"`
	TrcURL       FStr       `json:"trcUrl"`
}

// mgImg is one albumImgs entry. The size type selects the rendition of the cover.
type mgImg struct {
	Img FStr `json:"img"`
}

// mgListenItem is data.songItem of listen-url, the detail record. It is what
// upstream's resourceinfo.do used to supply, and it is where the per-word lyric
// url comes from - the search does not carry one.
type mgListenItem struct {
	SongID      FStr       `json:"songId"`
	CopyrightID FStr       `json:"copyrightId"`
	SongName    FStr       `json:"songName"`
	Album       FStr       `json:"album"`
	AlbumID     FStr       `json:"albumId"`
	Singer      FStr       `json:"singer"`
	Artists     []mgArtist `json:"artists"`
	Length      FStr       `json:"length"`
	TopQuality  FStr       `json:"topQuality"`
	AlbumImgs   []mgImg    `json:"albumImgs"`
	LrcURL      FStr       `json:"lrcUrl"`
	MrcURL      FStr       `json:"mrcUrl"`
	TrcURL      FStr       `json:"trcUrl"`
}

// mgListenData is the listen-url payload. url is the rendition to stream and is
// empty whenever the caller is not entitled to the toneFlag asked for; songItem
// is present regardless of tone, which is what makes this the detail endpoint.
type mgListenData struct {
	URL        FStr          `json:"url"`
	FormatType FStr          `json:"formatType"`
	SongItem   *mgListenItem `json:"songItem"`
}

func (it mgSearchItem) mrcURL() string {
	if v := string(it.MrcURL); v != "" {
		return v
	}
	return string(it.MrcURLSnake)
}

// mgFormatTypes is the desktop app's switch. ZQ is one spelling of the 24-bit
// tier, ZQ24 the other, and both land on it.
var mgFormatTypes = map[string]string{
	"PQ":   Q128,
	"HQ":   Q320,
	"SQ":   QFlac,
	"ZQ":   QFlac24,
	"ZQ24": QFlac24,
}

// mgQualities folds a rendition list into qualities, keeping one entry per tier.
// mg announces no size for most of its 128k rows, and that is fine: an empty
// size just means the platform did not say.
func mgQualities(fmts []mgFormat) []Quality {
	var out []Quality
	for _, f := range fmts {
		t := mgFormatTypes[string(f.FormatType)]
		if t == "" || contains(qualityTypes(out), t) {
			continue
		}
		out = append(out, Quality{Type: t, Size: f.SizeBytes()})
	}
	if len(out) == 0 {
		out = []Quality{{Type: Q128}}
	}
	return out
}

// mgSingers joins the artist list with 、, the same separator the other sources
// use, so the table and the relevance scorer see one shape.
func mgSingers(list []mgArtist) string {
	var names []string
	for _, a := range list {
		if v := strings.TrimSpace(string(a.Name)); v != "" {
			names = append(names, v)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return html.UnescapeString(strings.Join(names, "、"))
}

// mgSeconds normalizes a duration field. The search endpoint feeds it straight
// into formatPlayTime, which divides by 60, so it is seconds - but migu has been
// known to hand back milliseconds, and a 5:32 song must not read as 5533
// minutes. A value is taken as milliseconds only when it is implausibly long as
// seconds and plausible as milliseconds: dividing unconditionally would shrink a
// three-hour ambient track to 12 minutes.
func mgSeconds(v FInt) int {
	s := int64(v)
	if s < 0 {
		return 0
	}
	const maxSeconds = 3 * 3600
	if s > maxSeconds && s/1000 <= maxSeconds {
		s = s / 1000
	}
	return int(s)
}

// mgInterval parses listen-url's length, which is free text: "mm:ss" for a short
// song and "hh:mm:ss" once it passes the hour, sometimes with a ".000" tail.
func mgInterval(length string) int {
	parts := strings.FieldsFunc(length, func(r rune) bool { return r == ':' })
	// The last field can carry a ".000" tail, which truncates rather than parses.
	atoi := func(s string) int {
		s = strings.TrimSpace(s)
		if i := strings.IndexByte(s, '.'); i >= 0 {
			s = s[:i]
		}
		v, _ := strconv.Atoi(s)
		return v
	}
	switch len(parts) {
	case 2:
		return atoi(parts[0])*60 + atoi(parts[1])
	case 3:
		return atoi(parts[0])*3600 + atoi(parts[1])*60 + atoi(parts[2])
	}
	return 0
}

func (m *Migu) searchHeaders(ts, sign string) map[string]string {
	return map[string]string{
		"uiVersion":  mgUIVersion,
		"deviceId":   mgDeviceID,
		"timestamp":  ts,
		"sign":       sign,
		"channel":    mgChannel,
		"User-Agent": mgSearchUA,
	}
}

// Search calls the v3 search. The signature covers the raw keyword, not the
// encoded one: sign is md5(keyword + seed + app + deviceId + timestamp).
func (m *Migu) Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sign := md5Hex(keyword + mgSignSeed + mgSignApp + mgDeviceID + ts)
	u := fmt.Sprintf("%s?isCorrect=0&isCopyright=1&searchSwitch=%s&pageSize=%d&text=%s&pageNo=%d&sort=0&sid=USS",
		mgSearchURL, mgSearchSwitch, limit, url.QueryEscape(keyword), page)

	var resp struct {
		Code FStr `json:"code"`
		Info FStr `json:"info"`
		Song struct {
			TotalCount FStr             `json:"totalCount"`
			ResultList [][]mgSearchItem `json:"resultList"`
		} `json:"songResultData"`
	}
	if err := m.C.GetJSON(ctx, u, m.searchHeaders(ts, sign), &resp); err != nil {
		return nil, err
	}
	// The success code is the string "000000", not a number. Anything else is a
	// refusal and info carries the reason.
	if c := string(resp.Code); c != mgOK {
		return nil, fmt.Errorf("mg: search rejected code=%s %s", c, strings.TrimSpace(string(resp.Info)))
	}

	total, _ := strconv.Atoi(string(resp.Song.TotalCount))
	out := &SearchResult{
		Source: m.ID(), List: make([]*Song, 0),
		Total: total, Page: page, Limit: limit,
		AllPage: (total + limit - 1) / limit,
	}
	seen := map[string]bool{}
	for _, group := range resp.Song.ResultList {
		for _, it := range group {
			s := mgSearchSong(m.ID(), it)
			if s == nil || seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			out.List = append(out.List, s)
		}
	}
	return out, nil
}

func mgSearchSong(src string, it mgSearchItem) *Song {
	// Song.ID is the song id, because listen-url - and with it detail, lyric, pic
	// and audio - takes the song id and not the copyright id. A row without one
	// would be a reference nothing can resolve, so it is dropped.
	if string(it.SongID) == "" || string(it.Name) == "" {
		return nil
	}
	copyright := string(it.CopyrightID)
	// The search serves relative cover paths. Upstream only checks img3, which
	// misses a cover taken from img2 or img1, so the check follows the value that
	// was actually picked.
	img := string(it.Img3)
	if img == "" {
		img = string(it.Img2)
	}
	if img == "" {
		img = string(it.Img1)
	}
	if img != "" && !strings.Contains(img, "://") {
		img = mgCDN + img
	}
	siders := mgSingers(it.SingerList)
	if siders == "" {
		siders = html.UnescapeString(string(it.ArtistName))
	}
	// The page link needs the copyright id; without it the row still resolves,
	// there is just no public link to point at.
	page := ""
	if copyright != "" {
		page = mgPage + copyright
	}
	return &Song{
		Source:    src,
		ID:        string(it.SongID),
		Name:      html.UnescapeString(string(it.Name)),
		Singer:    siders,
		Album:     html.UnescapeString(string(it.Album)),
		AlbumID:   string(it.AlbumID),
		Duration:  mgSeconds(it.Duration),
		Pic:       img,
		Qualities: mgQualities(it.AudioFormats),
		MID:       copyright,
		PageURL:   page,
		LrcURL:    string(it.LrcURL),
		MrcURL:    it.mrcURL(),
		TrcURL:    string(it.TrcURL),
	}
}

// mgListenHeaders is the header set listen-url wants. The channel is what the
// gateway switches on; nothing else is required.
func mgListenHeaders() map[string]string {
	return map[string]string{
		"channel":    mgChannel,
		"User-Agent": mgListenUA,
	}
}

// listen asks the gateway for one rendition of one song. songID is the song id,
// never the copyright id - see the package comment. tone selects the rendition
// and also decides whether data.url is filled in.
func (m *Migu) listen(ctx context.Context, songID, tone string) (*mgListenData, error) {
	u := fmt.Sprintf("%s?netType=00&resourceType=2&songId=%s&toneFlag=%s",
		mgListenURL, url.QueryEscape(songID), tone)
	var resp struct {
		Code FStr          `json:"code"`
		Info FStr          `json:"info"`
		Data *mgListenData `json:"data"`
	}
	if err := m.C.GetJSON(ctx, u, mgListenHeaders(), &resp); err != nil {
		return nil, err
	}
	if c := string(resp.Code); c != mgOK {
		info := strings.TrimSpace(string(resp.Info))
		if c == mgNoResourceCode {
			return nil, fmt.Errorf("%w (%s)", errMiguNoResource, info)
		}
		return nil, fmt.Errorf("mg: listen-url rejected code=%s %s", c, info)
	}
	return resp.Data, nil
}

// mgIDError turns a listen-url failure into a message a user can act on. The id
// hint is appended only for the no-resource answer, which is the one case where
// a copyright id may have been passed where a song id was wanted; a network
// error gets its plain error.
func mgIDError(id string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errMiguNoResource) {
		return fmt.Errorf("mg: %s: %s", id, mgIDNote)
	}
	return err
}

// Detail hydrates a song the caller only knows by id. It asks for PQ, the one
// tier the gateway always hands over, and reads the metadata off songItem -
// including the per-word lyric url, which the search does not carry.
func (m *Migu) Detail(ctx context.Context, s *Song) error {
	if s.ID == "" {
		return fmt.Errorf("mg: no song id for %s", s.KeyString())
	}
	d, err := m.listen(ctx, s.ID, mgTone[Q128])
	if err != nil {
		return mgIDError(s.ID, err)
	}
	it := d.SongItem
	if it == nil || string(it.SongName) == "" {
		return fmt.Errorf("mg: listen-url returned no song data for %s", s.KeyString())
	}
	if s.Name == "" {
		s.Name = html.UnescapeString(string(it.SongName))
	}
	singers := mgSingers(it.Artists)
	if singers == "" {
		singers = html.UnescapeString(string(it.Singer))
	}
	if s.Singer == "" {
		s.Singer = singers
	}
	if s.Album == "" {
		s.Album = html.UnescapeString(string(it.Album))
	}
	if s.AlbumID == "" {
		s.AlbumID = string(it.AlbumID)
	}
	if s.ID == "" {
		s.ID = string(it.SongID)
	}
	if s.MID == "" {
		s.MID = string(it.CopyrightID)
	}
	// PageURL is built from the copyright id, which is the id the public site's
	// song pages are addressed by.
	if s.PageURL == "" && s.MID != "" {
		s.PageURL = mgPage + s.MID
	}
	if s.Duration == 0 {
		s.Duration = mgInterval(string(it.Length))
	}
	if s.Pic == "" && len(it.AlbumImgs) > 0 {
		s.Pic = string(it.AlbumImgs[0].Img)
	}
	if s.LrcURL == "" {
		s.LrcURL = string(it.LrcURL)
	}
	if s.MrcURL == "" {
		s.MrcURL = string(it.MrcURL)
	}
	if s.TrcURL == "" {
		s.TrcURL = string(it.TrcURL)
	}
	// A song known only by id has no rendition matrix. PQ is the tier the gateway
	// will always serve, so it is announced as the one rendition there is.
	if len(s.Qualities) == 0 {
		s.Qualities = []Quality{{Type: Q128}}
	}
	if s.Name == "" {
		return fmt.Errorf("mg: no title for %s", s.KeyString())
	}
	return nil
}

// Pic resolves the cover. It comes from albumImgs in the detail record, so a
// song that came out of a search already has one and needs no request.
func (m *Migu) Pic(ctx context.Context, s *Song) (string, error) {
	if s.Pic != "" {
		return s.Pic, nil
	}
	if err := m.Detail(ctx, s); err != nil {
		return "", err
	}
	if s.Pic == "" {
		return "", fmt.Errorf("mg: no cover for %s", s.KeyString())
	}
	return s.Pic, nil
}

// mgText fetches one lyric url. migu answers 404 for a large share of its lyric
// urls, and a 404 is a final answer rather than a hiccup, so it is not retried;
// anything else is.
func (m *Migu) mgText(ctx context.Context, u string) (string, error) {
	if u == "" {
		return "", nil
	}
	h := map[string]string{
		"Referer":    "https://app.c.nf.migu.cn/",
		"channel":    mgChannel,
		"User-Agent": mgLyricUA,
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := m.C.Do(ctx, http.MethodGet, u, h, nil)
		if err == nil {
			return string(resp.Body), nil
		}
		last = err
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			break
		}
	}
	return "", last
}

// Lyric prefers the mrc url, which carries per-word timings, and falls back to
// the plain lrc url. The trc url is the translated lyric and is independent.
func (m *Migu) Lyric(ctx context.Context, s *Song) (*Lyric, error) {
	if s.MrcURL == "" && s.LrcURL == "" {
		if err := m.Detail(ctx, s); err != nil {
			return nil, err
		}
	}
	if s.MrcURL != "" {
		text, err := m.mgText(ctx, s.MrcURL)
		if err != nil {
			return nil, err
		}
		if l := parseMiguMRC(DecryptMiguLyric(text)); l != nil {
			l.URL = s.MrcURL
			if trc, err := m.mgText(ctx, s.TrcURL); err == nil {
				l.Trans = strings.TrimSpace(trc)
			}
			return l, nil
		}
	}
	if s.LrcURL != "" {
		text, err := m.mgText(ctx, s.LrcURL)
		if err != nil {
			return nil, err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("mg: lyric not found for %s", s.KeyString())
		}
		return &Lyric{Plain: text, Raw: text, Format: "lrc", URL: s.LrcURL}, nil
	}
	return nil, fmt.Errorf("mg: no lyric url for %s", s.KeyString())
}

// MusicURL resolves one rendition. The gateway answers code 000000 with an empty
// url instead of refusing, so the only way to learn that a tier is unavailable is
// to ask for it; the ladder is walked down from the tier asked for, stopping at
// the first one that answers. The compromise is reported in Note rather than
// failing, because a requested tier the platform will not serve is not a broken
// reference - the song is downloadable, just not at that fidelity.
func (m *Migu) MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error) {
	if s.ID == "" {
		return nil, fmt.Errorf("mg: no song id for %s", s.KeyString())
	}
	tone := mgTone[quality.Type]
	if tone == "" {
		// auto, empty, or a tier mg does not name: start at the one that always
		// answers. Auto stays silent on the way down, since the user asked for
		// whatever there is.
		tone = mgTone[Q128]
	}
	var data *mgListenData
	var got string
	var lastErr error
	for _, t := range mgLadderFrom(tone) {
		d, err := m.listen(ctx, s.ID, t)
		if err != nil {
			lastErr = err
			continue
		}
		if d != nil && string(d.URL) != "" {
			data, got = d, t
			break
		}
	}
	if data == nil {
		if lastErr != nil {
			return nil, mgIDError(s.ID, lastErr)
		}
		return nil, fmt.Errorf("mg: %s has no playable rendition", s.KeyString())
	}
	pl := &PlayerURL{
		URL:     string(data.URL),
		Quality: mgQualityForTone(got),
		Source:  m.ID(),
	}
	if got != tone {
		pl.Note = fmt.Sprintf("asked %s, served %s - migu gives out no higher tier without a login",
			mgQualityForTone(tone), pl.Quality)
	}
	return pl, nil
}

// ------------------------------------------------------------- lyric codec

const (
	mgTeaDelta int64 = 2654435769
	mgTeaMin   int   = 32
)

// mgTeaKey is the nine 64-bit words mg uses as a TEA key, verbatim from
// musicSdk/mg/utils/mrc.js. Only the first four are ever indexed.
var mgTeaKey = [9]int64{
	27303562373562475,
	18014862372307051,
	22799692160172081,
	34058940340699235,
	30962724186095721,
	27303523720101991,
	27303523720101998,
	31244139033526382,
	28992395054481524,
}

// DecryptMiguLyric runs mg's TEA over the hex body of an mrcUrl response and
// returns UTF-8 text. Bodies under 32 chars and bodies that are not hex at all
// come back unchanged: the first is how mg marks an unencrypted payload, the
// second is a plain lrcUrl served from an mrcUrl slot.
func DecryptMiguLyric(data string) string {
	s := strings.TrimSpace(data)
	if len(s) < mgTeaMin {
		return s
	}
	n := len(s) / 16
	if n == 0 {
		return s
	}
	d := make([]int64, n)
	for i := 0; i < n; i++ {
		v, err := strconv.ParseUint(s[i*16:(i+1)*16], 16, 64)
		if err != nil {
			return s
		}
		d[i] = int64(v)
	}
	mgTeaDecrypt(d, mgTeaKey[:])
	return mgLongsToUTF16(d)
}

// mgTeaDecrypt ports teaDecrypt exactly. The JavaScript does arbitrary-precision
// arithmetic and wraps every step back into signed 64-bit with toLong; Go
// int64 arithmetic wraps modulo 2^64 on its own and its >> is arithmetic, so the
// two agree and no explicit wrapping is needed.
func mgTeaDecrypt(d []int64, key []int64) {
	n := int64(len(d))
	if n < 1 {
		return
	}
	j2 := d[0]
	j3 := (6 + 52/n) * mgTeaDelta
	for {
		j4 := j3
		if j4 == 0 {
			break
		}
		j5 := int64(int64(3) & (j4 >> 2))
		j6 := n
		for {
			j6--
			if j6 > 0 {
				j7 := d[j6-1]
				i := int(j6)
				a := (j2 ^ j4) + (j7 ^ key[(j6&3)^j5])
				b := ((j7 >> 5) ^ (j2 << 2)) + ((j2 >> 3) ^ (j7 << 4))
				j2 = d[i] - (a ^ b)
				d[i] = j2
			} else {
				break
			}
		}
		j8 := d[n-1]
		// The inner loop has driven j6 to zero, so its key index is j5.
		a := (key[j5] ^ j8) + (j2 ^ j4)
		b := ((j8 >> 5) ^ (j2 << 2)) + ((j2 >> 3) ^ (j8 << 4))
		j2 = d[0] - (a ^ b)
		d[0] = j2
		j3 = j4 - mgTeaDelta
	}
}

// mgLongsToUTF16 turns the words back into text: eight little-endian bytes each,
// read as UTF-16LE. Decoding the whole buffer at once means a surrogate pair
// that straddles a word boundary still assembles.
func mgLongsToUTF16(d []int64) string {
	u := make([]uint16, 0, len(d)*4)
	buf := make([]byte, 8)
	for _, v := range d {
		for i := 0; i < 8; i++ {
			buf[i] = byte(v)
			v >>= 8
		}
		for i := 0; i < 8; i += 2 {
			u = append(u, uint16(buf[i])|uint16(buf[i+1])<<8)
		}
	}
	return string(utf16.Decode(u))
}

// ------------------------------------------------------------- mrc parse

var (
	// mgMrcLineTime is the line's absolute start and duration, in milliseconds.
	mgMrcLineTime = regexp.MustCompile(`^\s*\[(\d+),\d+\]`)
	// mgMrcWordTimeAll is the per-word marker, capturing start and duration.
	mgMrcWordTimeAll = regexp.MustCompile(`\((\d+),(\d+)\)`)
	// mgMrcWordTime is the same marker used as a split point.
	mgMrcWordTime = regexp.MustCompile(`\(\d+,\d+\)`)
)

// parseMiguMRC ports the mrc parser: "[ms,dur]" line heads, "(ms,dur)" per-word
// markers, and relative word starts derived from the line start. Plain holds
// the lyric with the markers stripped; Raw holds them, which is what the
// karaoke view needs.
func parseMiguMRC(src string) *Lyric {
	e := strings.ReplaceAll(src, "\r", "")
	var plain, synced []string
	for _, line := range strings.Split(e, "\n") {
		if len(line) < 6 {
			continue
		}
		m := mgMrcLineTime.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		start := krcInt(line[m[2]:m[3]])
		tag := mgMrcTag(start)
		words := line[:m[0]] + line[m[1]:]
		plain = append(plain, tag+mgMrcWordTime.ReplaceAllString(words, ""))

		times := mgMrcWordTimeAll.FindAllStringSubmatch(words, -1)
		if len(times) == 0 {
			continue
		}
		pieces := mgMrcWordTime.Split(words, -1)
		var sb strings.Builder
		for i, t := range times {
			st, _ := strconv.Atoi(t[1])
			dur, _ := strconv.Atoi(t[2])
			fmt.Fprintf(&sb, "<%d,%d>", st-start, dur)
			if i < len(pieces) {
				sb.WriteString(pieces[i])
			}
		}
		synced = append(synced, tag+sb.String())
	}
	if len(plain) == 0 {
		return nil
	}
	return &Lyric{
		Plain:  strings.TrimSpace(strings.Join(plain, "\n")),
		Raw:    strings.TrimSpace(strings.Join(synced, "\n")),
		Format: "lrc",
	}
}

// mgMrcTag renders milliseconds as "[mm:ss.mmm]", which is the LRC form.
func mgMrcTag(ms int) string {
	sec := ms / 1000
	return fmt.Sprintf("[%02d:%02d.%03d]", sec/60, sec%60, ms%1000)
}
