package lx

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

/*
NetEase, source id "wy".

Search, detail and lyric are implemented in the app itself; the audio URL goes
through the User API plugin (`action: "musicUrl"`), so MusicURL here uses
netease's public cheetah endpoint.

  search  POST http://interface.music.163.com/eapi/batch      (form: eapi params)
  detail  POST https://music.163.com/weapi/v3/song/detail     (form: weapi params)
  lyric   POST https://interface3.music.163.com/eapi/song/lyric/v1
  url     GET  https://music.163.com/api/cheetah/player/url
*/

const (
	neteaseBatchURL   = "http://interface.music.163.com/eapi/batch"
	neteaseDetailURL  = "https://music.163.com/weapi/v3/song/detail"
	neteaseLyricURL   = "https://interface3.music.163.com/eapi/song/lyric/v1"
	neteaseCheetahURL = "https://music.163.com/api/cheetah/player/url"
	neteaseUA         = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36"
	neteaseOrigin     = "https://music.163.com"
)

// neteaseSong is one song record as returned by simpleSongData / song/detail.
type neteaseSong struct {
	ID   FStr `json:"id"`
	Name FStr `json:"name"`
	DT   FInt `json:"dt"`
	Ar   []struct {
		Name FStr `json:"name"`
	} `json:"ar"`
	Al struct {
		Name   FStr `json:"name"`
		ID     FStr `json:"id"`
		PicURL FStr `json:"picUrl"`
	} `json:"al"`
	Privilege struct {
		MaxBrLevel FStr `json:"maxBrLevel"`
		Maxbr      FInt `json:"maxbr"`
	} `json:"privilege"`
	Hr struct {
		Size FInt `json:"size"`
	} `json:"hr"`
	Sq struct {
		Size FInt `json:"size"`
	} `json:"sq"`
	H struct {
		Size FInt `json:"size"`
	} `json:"h"`
	L struct {
		Size FInt `json:"size"`
	} `json:"l"`
}

// Netease implements Provider for netease music.
type Netease struct{ C *Client }

func (n *Netease) ID() string   { return "wy" }
func (n *Netease) Name() string { return "网易云音乐" }

// Anonymous audio: cheetah answers without a url - checked on six free tracks,
// all six came back empty, so it is the shape of the answer and not a paywall.
func (n *Netease) AnonymousAudio() bool { return false }

func (n *Netease) headers() map[string]string {
	return map[string]string{
		"User-Agent": neteaseUA,
		"origin":     neteaseOrigin,
		"Origin":     neteaseOrigin,
	}
}

// Search posts through the eapi/batch gateway with the app's signing scheme.
func (n *Netease) Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	if page <= 0 {
		page = 1
	}
	form, err := NeteaseEAPIForm("/api/search/song/list/page", map[string]any{
		"keyword":     keyword,
		"needCorrect": "1",
		"channel":     "typing",
		"offset":      limit * (page - 1),
		"scene":       "normal",
		"total":       page == 1,
		"limit":       limit,
	})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code FInt `json:"code"`
		Data struct {
			TotalCount FInt `json:"totalCount"`
			Resources  []struct {
				BaseInfo struct {
					SimpleSongData neteaseSong `json:"simpleSongData"`
				} `json:"baseInfo"`
			} `json:"resources"`
		} `json:"data"`
	}
	if err := n.C.PostForm(ctx, neteaseBatchURL, n.headers(), form, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("wy: search code=%d", resp.Code)
	}

	total := int(resp.Data.TotalCount)
	out := &SearchResult{
		Source: n.ID(), List: make([]*Song, 0, len(resp.Data.Resources)),
		Total: total, Page: page, Limit: limit,
		AllPage: (total + limit - 1) / limit,
	}
	seen := map[string]bool{}
	for _, r := range resp.Data.Resources {
		s := n.toSong(r.BaseInfo.SimpleSongData)
		if s == nil || seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		out.List = append(out.List, s)
	}
	return out, nil
}

// toSong mirrors handleResult(): flac24bit is always pushed, then the
// maxbr fall-through switch, and the whole list is reversed.
func (n *Netease) toSong(e neteaseSong) *Song {
	if string(e.ID) == "" {
		return nil
	}
	var t []Quality
	if e.Hr.Size > 0 {
		t = append(t, Quality{Type: QFlac24, Size: int64(e.Hr.Size)})
	}
	switch e.Privilege.Maxbr {
	case 999000:
		if e.Sq.Size > 0 {
			t = append(t, Quality{Type: QFlac, Size: int64(e.Sq.Size)})
		}
		fallthrough
	case 320000:
		if e.H.Size > 0 {
			t = append(t, Quality{Type: Q320, Size: int64(e.H.Size)})
		}
		fallthrough
	case 192000, 128000:
		if e.L.Size > 0 {
			t = append(t, Quality{Type: Q128, Size: int64(e.L.Size)})
		}
	}
	if len(t) == 0 {
		t = []Quality{{Type: Q128}}
	}

	singers := make([]string, 0, len(e.Ar))
	for _, a := range e.Ar {
		if v := string(a.Name); v != "" {
			singers = append(singers, v)
		}
	}
	return &Song{
		Source:    n.ID(),
		ID:        string(e.ID),
		Name:      string(e.Name),
		Singer:    strings.Join(singers, "、"),
		Album:     string(e.Al.Name),
		AlbumID:   string(e.Al.ID),
		Duration:  int(e.DT / 1000),
		Pic:       string(e.Al.PicURL),
		Qualities: t,
		MID:       string(e.ID),
		PageURL:   "https://music.163.com/#/song?id=" + string(e.ID),
	}
}

// Detail refetches one song through weapi/v3/song/detail.
func (n *Netease) Detail(ctx context.Context, s *Song) error {
	form, err := NeteaseWeAPIForm(map[string]string{
		"c":   fmt.Sprintf(`[{"id":%s}]`, s.ID),
		"ids": fmt.Sprintf(`[%s]`, s.ID),
	})
	if err != nil {
		return err
	}
	var resp struct {
		Code  FInt          `json:"code"`
		Songs []neteaseSong `json:"songs"`
	}
	h := map[string]string{
		"User-Agent": neteaseUA,
		"Referer":    "https://music.163.com/song?id=" + s.ID,
		"Origin":     neteaseOrigin,
	}
	if err := n.C.PostForm(ctx, neteaseDetailURL, h, form, &resp); err != nil {
		return err
	}
	if resp.Code != 200 || len(resp.Songs) == 0 {
		return fmt.Errorf("wy: song detail unavailable for %s", s.KeyString())
	}
	e := resp.Songs[0]
	if s.Name == "" {
		s.Name = string(e.Name)
	}
	if s.Album == "" {
		s.Album = string(e.Al.Name)
	}
	if s.Pic == "" {
		s.Pic = string(e.Al.PicURL)
	}
	return nil
}

// Pic returns the album cover carried by the search result.
func (n *Netease) Pic(ctx context.Context, s *Song) (string, error) {
	if s.Pic != "" {
		return s.Pic, nil
	}
	if err := n.Detail(ctx, s); err != nil || s.Pic == "" {
		return "", fmt.Errorf("wy: no cover for %s", s.KeyString())
	}
	return s.Pic, nil
}

// Lyric posts through /eapi/song/lyric/v1. The response already holds plain LRC
// text with [mm:ss.xx] tags.
func (n *Netease) Lyric(ctx context.Context, s *Song) (*Lyric, error) {
	form, err := NeteaseEAPIForm("/api/song/lyric/v1", map[string]any{
		"id": s.ID, "cp": false, "tv": 0, "lv": 0, "rv": 0,
		"kv": 0, "yv": 0, "ytv": 0, "yrv": 0,
	})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code FInt `json:"code"`
		Lrc  struct {
			Lyric FStr `json:"lyric"`
		} `json:"lrc"`
		Tlyric struct {
			Lyric FStr `json:"lyric"`
		} `json:"tlyric"`
		Romalrc struct {
			Lyric FStr `json:"lyric"`
		} `json:"romalrc"`
		Yrc struct {
			Lyric FStr `json:"lyric"`
		} `json:"yrc"`
		Ytlrc struct {
			Lyric FStr `json:"lyric"`
		} `json:"ytlrc"`
		Yromalrc struct {
			Lyric FStr `json:"lyric"`
		} `json:"yromalrc"`
	}
	if err := n.C.PostForm(ctx, neteaseLyricURL, n.headers(), form, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("wy: lyric code=%d", resp.Code)
	}
	plain := pickLyric(string(resp.Lrc.Lyric), string(resp.Yrc.Lyric))
	trans := pickLyric(string(resp.Tlyric.Lyric), string(resp.Ytlrc.Lyric))
	rom := pickLyric(string(resp.Romalrc.Lyric), string(resp.Yromalrc.Lyric))
	if plain == "" {
		return nil, fmt.Errorf("wy: lyric not found for %s", s.KeyString())
	}
	return &Lyric{
		Plain:  normalizeNeteaseLRC(plain),
		Trans:  normalizeNeteaseLRC(trans),
		Roman:  normalizeNeteaseLRC(rom),
		Format: "lrc", Raw: plain, URL: neteaseLyricURL,
	}, nil
}

// pickLyric prefers the plain LRC payload and falls back to the dynamic lyric,
// converting the JSON lines into LRC as needed.
func pickLyric(plain, yrc string) string {
	if strings.TrimSpace(plain) != "" {
		return neteaseYrcToLRC(plain)
	}
	return neteaseYrcToLRC(yrc)
}

// neteaseYrcLine is one line of netease's dynamic lyric ("yrc") payload.
type neteaseYrcLine struct {
	T int `json:"t"`
	C []struct {
		TX string `json:"tx"`
	} `json:"c"`
}

// neteaseYrcToLRC turns netease's dynamic lyric lines into plain LRC. Netease
// mixes these JSON lines into the plain payload, so everything else is passed
// through untouched.
func neteaseYrcToLRC(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "{") {
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		var d neteaseYrcLine
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			continue
		}
		var tx strings.Builder
		for _, c := range d.C {
			tx.WriteString(c.TX)
		}
		if strings.TrimSpace(tx.String()) == "" {
			continue
		}
		b.WriteString("[" + FormatLRCTime(int64(d.T)) + "]" + tx.String() + "\n")
	}
	return strings.TrimSpace(b.String())
}

// neteaseTagThree is the [mm:ss:xx] tag shape netease sends back.
var neteaseTagThree = regexp.MustCompile(`\[(\d{2}:\d{2}):(\d{2,3})\]`)

// normalizeNeteaseLRC applies the app's [mm:ss:xx] -> [mm:ss.xx] rewrite.
func normalizeNeteaseLRC(s string) string {
	if s == "" {
		return ""
	}
	return strings.TrimSpace(neteaseTagThree.ReplaceAllString(s, "[$1.$2]"))
}

// MusicURL asks the cheetah endpoint, which serves anonymous playback URLs for
// most non-VIP songs.
func (n *Netease) MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error) {
	br := 128000
	switch quality.Type {
	case Q320:
		br = 320000
	case QFlac:
		br = 3200000
	case QFlac24:
		br = 9990000
	case "", QAuto, QBestKnown:
		switch s.BestQuality() {
		case Q320:
			br = 320000
		case QFlac:
			br = 3200000
		case QFlac24:
			br = 9990000
		}
	}
	u := fmt.Sprintf("%s?ids=[%s]&br=%d", neteaseCheetahURL, s.ID, br)
	var resp struct {
		Code FInt `json:"code"`
		Data []struct {
			URL  FStr `json:"url"`
			Size FInt `json:"size"`
			Br   FInt `json:"br"`
		} `json:"data"`
	}
	if err := n.C.GetJSON(ctx, u, n.headers(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 || len(resp.Data) == 0 || string(resp.Data[0].URL) == "" {
		return nil, fmt.Errorf("wy: no url for %s (cheetah returns nothing for VIP/pay songs; the desktop app resolves these via its User API plugin)", s.KeyString())
	}
	ext := ".mp3"
	if quality.Type == QFlac || quality.Type == QFlac24 {
		ext = ".flac"
	}
	return &PlayerURL{
		URL: string(resp.Data[0].URL), Quality: quality.Type, Source: n.ID(),
		Size: int64(resp.Data[0].Size), Duration: s.Duration,
		FileName: s.Name + ext,
		Headers:  map[string]string{"Referer": "https://music.163.com/"},
	}, nil
}
