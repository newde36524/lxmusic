package lx

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
)

/*
Tencent, source id "tx".

Search and lyric are implemented in the app; the audio URL is handed to the
User API plugin (`action: "musicUrl"`), which is why the bundle carries a full
`sign` implementation but no URL endpoint at all.

  search  POST https://u.y.qq.com/cgi-bin/musics.fcg?sign=<TencentSign(body)>
  lyric   POST https://u.y.qq.com/cgi-bin/musicu.fcg   (no sign)
  url     GET  https://u.qq.com/cgi-bin/musicurl.fcgi   -> ws.stream.qqmusic.qq.com
*/

const (
	txSearchURL = "https://u.y.qq.com/cgi-bin/musics.fcg"
	txMusicuURL = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	txFcgiURL   = "https://u.qq.com/cgi-bin/musicurl.fcgi"
	txPicAlbum  = "https://y.gtimg.cn/music/photo_new/T002R500x500M000%s.jpg"
	txPicSinger = "https://y.gtimg.cn/music/photo_new/T001R500x500M000%s.jpg"
	txPage      = "https://y.qq.com/n/yqq/song/%s.html"
	txUA        = "QQMusic 14090508(android 12)"
	txGuid      = "1F70E520B2EAA7D25E11760783C53CA9"
	txWid       = "7223299733393904640"
)

// Tencent implements Provider for Tencent music.
type Tencent struct{ C *Client }

func (t *Tencent) ID() string   { return "tx" }
func (t *Tencent) Name() string { return "QQ音乐" }

// Anonymous audio: the url carries a vkey, and getting one needs a logged-in
// uin.
func (t *Tencent) AnonymousAudio() bool { return false }

// txItem is one entry of data.song.list.
type txItem struct {
	ID       FStr `json:"id"`
	Mid      FStr `json:"mid"`
	Title    FStr `json:"title"`
	Interval FInt `json:"interval"`
	Album    struct {
		Name FStr `json:"name"`
		Mid  FStr `json:"mid"`
	} `json:"album"`
	Singer []struct {
		Name FStr `json:"name"`
		Mid  FStr `json:"mid"`
	} `json:"singer"`
	File struct {
		MediaMid   FStr `json:"media_mid"`
		Size128mp3 FInt `json:"size_128mp3"`
		Size320mp3 FInt `json:"size_320mp3"`
		SizeFlac   FInt `json:"size_flac"`
		SizeHires  FInt `json:"size_hires"`
	} `json:"file"`
}

// txHolder tolerates both the flat data.song.list shape and the data.body
// wrapper that the android client receives.
type txHolder struct {
	List []txItem `json:"list"`
	Sum  FInt     `json:"sum"`
}

// searchID is a fresh searchid: 32 uppercase hex chars plus 5 digits.
func txSearchID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b)) + strconv.Itoa(rand.Intn(100000))
}

// Search calls DoSearchForQQMusicDesktop with the recovered sign scheme.
func (t *Tencent) Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	body := map[string]any{
		"comm": map[string]any{
			"_channelid":                  "0",
			"_os_version":                 "6.2.9200-2",
			"ct":                          "19",
			"cv":                          "2151",
			"guid":                        txGuid,
			"patch":                       "118",
			"psrf_access_token_expiresAt": 0,
			"psrf_qqaccess_token":         "",
			"psrf_qqopenid":               "",
			"psrf_qqunionid":              "",
			"tmeAppID":                    "qqmusic",
			"tmeLoginType":                0,
			"uin":                         "0",
			"wid":                         txWid,
		},
		"music.search.SearchCgiService": map[string]any{
			"module": "music.search.SearchCgiService",
			"method": "DoSearchForQQMusicDesktop",
			"param": map[string]any{
				"grp":          1,
				"num_per_page": limit,
				"page_num":     page,
				"query":        keyword,
				"remoteplace":  "txt.newclient.top",
				"search_type":  0,
				"searchid":     txSearchID(),
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	sign := TencentSign(payload)

	var resp struct {
		Code FInt `json:"code"`
		Svc  struct {
			Code FInt `json:"code"`
			Data struct {
				Song struct {
					List []txItem `json:"list"`
					Sum  FInt     `json:"sum"`
				} `json:"song"`
				Meta struct {
					Sum FInt `json:"sum"`
				} `json:"meta"`
				Body struct {
					Song struct {
						List []txItem `json:"list"`
						Sum  FInt     `json:"sum"`
					} `json:"song"`
				} `json:"body"`
			} `json:"data"`
		} `json:"music.search.SearchCgiService"`
	}
	h := map[string]string{"User-Agent": txUA, "Referer": "https://y.qq.com"}
	if err := t.C.PostJSON(ctx, txSearchURL+"?sign="+sign, h, body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 || resp.Svc.Code != 0 {
		return nil, fmt.Errorf("tx: search code=%d/%d", resp.Code, resp.Svc.Code)
	}
	list := resp.Svc.Data.Song.List
	if len(list) == 0 {
		list = resp.Svc.Data.Body.Song.List
	}
	total := int(resp.Svc.Data.Song.Sum)
	if total == 0 {
		total = int(resp.Svc.Data.Meta.Sum)
	}

	out := &SearchResult{
		Source: t.ID(), List: make([]*Song, 0, len(list)),
		Total: total, Page: page, Limit: limit,
		AllPage: (total + limit - 1) / limit,
	}
	seen := map[string]bool{}
	for _, it := range list {
		s := t.toSong(it)
		if s == nil || seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		out.List = append(out.List, s)
	}
	return out, nil
}

// toSong mirrors handleResult(), including the T001/T002 cover decision.
func (t *Tencent) toSong(e txItem) *Song {
	if string(e.File.MediaMid) == "" {
		return nil
	}
	var q []Quality
	if e.File.Size128mp3 > 0 {
		q = append(q, Quality{Type: Q128, Size: int64(e.File.Size128mp3)})
	}
	if e.File.Size320mp3 > 0 {
		q = append(q, Quality{Type: Q320, Size: int64(e.File.Size320mp3)})
	}
	if e.File.SizeFlac > 0 {
		q = append(q, Quality{Type: QFlac, Size: int64(e.File.SizeFlac)})
	}
	if e.File.SizeHires > 0 {
		q = append(q, Quality{Type: QFlac24, Size: int64(e.File.SizeHires)})
	}
	if len(q) == 0 {
		q = []Quality{{Type: Q128}}
	}

	singers := make([]string, 0, len(e.Singer))
	for _, a := range e.Singer {
		if v := string(a.Name); v != "" {
			singers = append(singers, v)
		}
	}

	albumMid := string(e.Album.Mid)
	var pic string
	if albumMid != "" && albumMid != "空" {
		pic = fmt.Sprintf(txPicAlbum, albumMid)
	} else if len(e.Singer) > 0 {
		pic = fmt.Sprintf(txPicSinger, string(e.Singer[0].Mid))
	}

	return &Song{
		Source:    t.ID(),
		ID:        string(e.File.MediaMid),
		Name:      string(e.Title),
		Singer:    strings.Join(singers, "、"),
		Album:     string(e.Album.Name),
		AlbumID:   albumMid,
		Duration:  int(e.Interval),
		Pic:       pic,
		Qualities: q,
		MID:       string(e.Mid),
		NumericID: string(e.ID),
		PageURL:   fmt.Sprintf(txPage, string(e.Mid)),
	}
}

// Detail is a no-op: the search payload already carries the full rendition
// matrix, and the per-song detail endpoint is the one the User API plugin owns.
func (t *Tencent) Detail(ctx context.Context, s *Song) error { return nil }

// Pic prefers the album cover and falls back to the first singer's headshot.
func (t *Tencent) Pic(ctx context.Context, s *Song) (string, error) {
	if s.Pic != "" {
		return s.Pic, nil
	}
	if s.AlbumID != "" && s.AlbumID != "空" {
		s.Pic = fmt.Sprintf(txPicAlbum, s.AlbumID)
		return s.Pic, nil
	}
	if s.MID != "" {
		s.Pic = fmt.Sprintf(txPicSinger, s.MID)
		return s.Pic, nil
	}
	return "", fmt.Errorf("tx: no album or singer mid for %s", s.KeyString())
}

// Lyric calls music.musichallSong.PlayLyricInfo, which needs no sign. The
// service wants the numeric songId, not the songmid.
func (t *Tencent) Lyric(ctx context.Context, s *Song) (*Lyric, error) {
	songID := s.NumericID
	if songID == "" {
		songID = s.MID
	}
	if songID == "" {
		songID = s.ID
	}
	// The service wants the id as a JSON number, not a string: a string form
	// is answered with code 10006.
	var songVal any = songID
	if n, err := strconv.ParseInt(songID, 10, 64); err == nil {
		songVal = n
	}
	body := map[string]any{
		"comm": map[string]any{"ct": "19", "cv": "1859", "uin": "0"},
		"req": map[string]any{
			"method": "GetPlayLyricInfo",
			"module": "music.musichallSong.PlayLyricInfo",
			"param": map[string]any{
				"format": "json", "crypt": 1, "ct": 19, "cv": 1873,
				"interval": 0, "lrc_t": 0, "qrc": 1, "qrc_t": 0,
				"roma": 1, "roma_t": 0, "songID": songVal,
				"trans": 1, "trans_t": 0, "type": -1,
			},
		},
	}
	var resp struct {
		Code FInt `json:"code"`
		Req  struct {
			Code FInt `json:"code"`
			Data struct {
				Lyric FStr `json:"lyric"`
				Trans FStr `json:"trans"`
				Roma  FStr `json:"roma"`
			} `json:"data"`
		} `json:"req"`
	}
	if err := t.C.PostJSON(ctx, txMusicuURL, map[string]string{
		"referer":    "https://y.qq.com",
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
	}, body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 || resp.Req.Code != 0 {
		return nil, fmt.Errorf("tx: lyric code=%d/%d", resp.Code, resp.Req.Code)
	}
	// Each field comes back as a hex-encoded blob that must run through
	// Tencent's private 64-bit block cipher before it is a zlib stream, and
	// the result is a QrcInfos karaoke payload that has to be reduced to LRC.
	plain := TencentDecryptLRC(string(resp.Req.Data.Lyric))
	trans := TencentDecryptLRC(string(resp.Req.Data.Trans))
	roman := TencentDecryptLRC(string(resp.Req.Data.Roma))
	if plain == "" {
		return nil, fmt.Errorf("tx: lyric not found for %s", s.KeyString())
	}
	return &Lyric{
		Plain:  plain,
		Trans:  trans,
		Roman:  roman,
		Format: "lrc", Raw: plain, URL: txMusicuURL,
	}, nil
}

// MusicURL resolves a stream URL through musicurl.fcgi. Tencent rotates this
// endpoint regularly and usually needs a logged-in uin to issue a vkey, which
// the desktop app gets from its User API plugin; this is the anonymous path.
func (t *Tencent) MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error) {
	bitrate := int64(128)
	switch quality.Type {
	case Q320:
		bitrate = 320
	case QFlac, QFlac24:
		bitrate = 2000
	case "":
		if b := s.BestQuality(); b == Q320 {
			bitrate = 320
		} else if b == QFlac || b == QFlac24 {
			bitrate = 2000
		}
	}
	u := fmt.Sprintf("%s?t=ympage&g_tk=5381&loginUin=0&hostUin=0&songmid=%s&format=json", txFcgiURL, urlQueryEscape(s.ID))
	var resp struct {
		Midurlinfo []struct {
			Filename FStr `json:"filename"`
			Vkey     FStr `json:"vkey"`
			Bitrate  FInt `json:"bitrate"`
			Filesize FInt `json:"filesize"`
			Fid      FStr `json:"fid"`
		} `json:"midurlinfo"`
		Playurl struct {
			Vkey FStr `json:"vkey"`
		} `json:"playurl"`
	}
	if err := t.C.GetJSON(ctx, u, map[string]string{"Referer": "https://y.qq.com"}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Midurlinfo) == 0 {
		return nil, fmt.Errorf("tx: no midurlinfo for %s (musicurl.fcgi usually needs a logged-in uin; the desktop app resolves this via its User API plugin)", s.KeyString())
	}
	entry := resp.Midurlinfo[0]
	if string(entry.Filename) == "" || string(entry.Vkey) == "" {
		return nil, fmt.Errorf("tx: no vkey for %s (song needs login or VIP)", s.KeyString())
	}
	stream := fmt.Sprintf("https://ws.stream.qqmusic.qq.com/%s?fromtag=1&guid=%s&vkey=%s&uin=0&br=%d&fid=%s",
		entry.Filename, txGuid, entry.Vkey, bitrate, entry.Fid)
	ext := ".mp3"
	if quality.Type == QFlac || quality.Type == QFlac24 {
		ext = ".flac"
	}
	return &PlayerURL{
		URL: stream, Quality: quality.Type, Source: t.ID(),
		Size: int64(entry.Filesize), Duration: s.Duration, FileName: s.Name + ext,
		Headers: map[string]string{"Referer": "https://y.qq.com"},
	}, nil
}

// urlQueryEscape keeps songmids URL-safe without percent-encoding the slashes.
func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}
