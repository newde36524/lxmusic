package lx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

/*
Kugou, source id "kg".

All endpoints below are lifted from dist/renderer.js of lx-music-desktop
2.12.6. Field names are case sensitive and were checked against the live API
(周杰伦 - 晴天, 2026-10-09).

  search   GET  http://songsearch.kugou.com/song_search_v2
  detail   POST http://gateway.kugou.com/v2/album_audio/audio  rendition matrix by hash
  url      POST http://media.store.kugou.com/v1/get_res_privilege  (returns no audio url)
  pic      POST http://media.store.kugou.com/v1/get_res_privilege  (same call, info.image)
  lyric    GET  http://lyrics.kugou.com/search  -> http://lyrics.kugou.com/download
*/

const (
	kugouSearch    = "http://songsearch.kugou.com/song_search_v2"
	kugouGateway   = "http://gateway.kugou.com/v2/album_audio/audio"
	kugouPrivilege = "http://media.store.kugou.com/v1/get_res_privilege"
	kugouLyricFind = "http://lyrics.kugou.com/search"
	kugouLyricDL   = "http://lyrics.kugou.com/download"
)

var kugouHeaders = map[string]string{
	"KG-RC":      "1",
	"KG-THash":   "expand_search_manager.cpp:852736169:451",
	"User-Agent": "KuGou2012-9020-ExpandSearchManager",
}

// Kugou implements Provider for kugou music.
type Kugou struct{ C *Client }

func (k *Kugou) ID() string   { return "kg" }
func (k *Kugou) Name() string { return "酷狗音乐" }

// Kugou never returns an audio url - see MusicURL - so no kugou row is
// pickable in the search table.
func (k *Kugou) AnonymousAudio() bool { return false }

// ------------------------------------------------------------- search

type kugouItem struct {
	Audioid     FStr `json:"Audioid"`
	MixSongID   FStr `json:"MixSongID"`
	AlbumID     FStr `json:"AlbumID"`
	AlbumName   FStr `json:"AlbumName"`
	OriSongName FStr `json:"OriSongName"`
	Suffix      FStr `json:"Suffix"`
	SingerName  FStr `json:"SingerName"`
	Duration    FInt `json:"Duration"`
	Privilege   FInt `json:"Privilege"`

	FileSize      FInt `json:"FileSize"`
	FileHash      FStr `json:"FileHash"`
	HQFileSize    FInt `json:"HQFileSize"`
	HQFileHash    FStr `json:"HQFileHash"`
	SQFileSize    FInt `json:"SQFileSize"`
	SQFileHash    FStr `json:"SQFileHash"`
	ResFileSize   FInt `json:"ResFileSize"`
	ResFileHash   FStr `json:"ResFileHash"`
	SuperFileSize FInt `json:"SuperFileSize"`
	SuperFileHash FStr `json:"SuperFileHash"`

	Singers []struct {
		Name FStr `json:"name"`
	} `json:"Singers"`
	Grp []kugouItem `json:"Grp"`
}

// Search calls song_search_v2 and normalizes every hit.
func (k *Kugou) Search(ctx context.Context, keyword string, page, limit int) (*SearchResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	q := url.Values{}
	q.Set("platform", "AndroidFilter")
	q.Set("iscorrection", "1")
	q.Set("keyword", keyword)
	q.Set("hifiquality", "0")
	q.Set("pagesize", strconv.Itoa(limit))
	q.Set("PrivilegeFilter", "0")
	q.Set("page", strconv.Itoa(page))

	var resp struct {
		ErrorCode FInt `json:"error_code"`
		ErrorMsg  FStr `json:"error_msg"`
		Data      struct {
			Total FInt        `json:"total"`
			Lists []kugouItem `json:"lists"`
		} `json:"data"`
	}
	if err := k.C.GetJSON(ctx, kugouSearch+"?"+q.Encode(), kugouHeaders, &resp); err != nil {
		return nil, err
	}
	if resp.ErrorCode != 0 {
		return nil, fmt.Errorf("kugou search: error_code=%d %s", resp.ErrorCode, resp.ErrorMsg)
	}

	total := int(resp.Data.Total)
	out := &SearchResult{
		Source: k.ID(), List: make([]*Song, 0, len(resp.Data.Lists)),
		Total: total, Page: page, Limit: limit,
		AllPage: (total + limit - 1) / limit,
	}
	// The app deduplicates on Audioid+FileHash, exactly like handleResult().
	seen := map[string]bool{}
	for _, it := range resp.Data.Lists {
		items := append([]kugouItem{it}, it.Grp...)
		for _, j := range items {
			s := k.toSong(j)
			if s == nil || seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			out.List = append(out.List, s)
		}
	}
	return out, nil
}

// toSong mirrors the desktop app's filterData(): qualities come from the
// File*/HQFile*/SQFile*/ResFile* triples and the id is Audioid+FileHash.
func (k *Kugou) toSong(it kugouItem) *Song {
	hash := string(it.FileHash)
	if hash == "" {
		return nil
	}
	name := strings.TrimSpace(string(it.OriSongName))
	if suf := strings.TrimSpace(string(it.Suffix)); suf != "" {
		name += " " + suf
	}
	if name == "" {
		return nil
	}

	var singers []string
	for _, a := range it.Singers {
		if n := string(a.Name); n != "" && !contains(singers, n) {
			singers = append(singers, n)
		}
	}
	if len(singers) == 0 {
		singers = []string{strings.TrimSpace(string(it.SingerName))}
	}

	var qualities []Quality
	add := func(t string, size FInt, h FStr) {
		if size == 0 || h == "" {
			return
		}
		for _, q := range qualities {
			if q.Hash == string(h) {
				return
			}
		}
		qualities = append(qualities, Quality{Type: t, Size: int64(size), Hash: string(h)})
	}
	add(Q128, it.FileSize, it.FileHash)
	add(Q320, it.HQFileSize, it.HQFileHash)
	add(QFlac, it.SQFileSize, it.SQFileHash)
	add(QFlac24, it.ResFileSize, it.ResFileHash)
	add(Q320, it.SuperFileSize, it.SuperFileHash)
	if len(qualities) == 0 {
		return nil
	}

	return &Song{
		Source:    k.ID(),
		ID:        string(it.Audioid) + hash,
		Name:      name,
		Singer:    strings.Join(singers, "、"),
		Album:     string(it.AlbumName),
		AlbumID:   string(it.AlbumID),
		Duration:  int(it.Duration),
		Qualities: qualities,
		Hash:      hash,
		MID:       string(it.Audioid),
		PageURL:   "https://www.kugou.com/song/#hash=" + hash + "&album_id=" + string(it.AlbumID),
	}
}

// Detail refetches the song through the gateway endpoint, which returns the
// complete rendition matrix (per-bitrate hash + filesize) for the 128k hash.
// It is what the desktop app uses to refresh a song it already has.
func (k *Kugou) Detail(ctx context.Context, s *Song) error {
	hash := s.Hash
	if hash == "" {
		return fmt.Errorf("kg: no hash for %s", s.KeyString())
	}
	body := map[string]any{
		"area_code":       "1",
		"show_privilege":  1,
		"show_album_info": "1",
		"is_publish":      "",
		"appid":           1005,
		"clientver":       11451,
		"mid":             "1",
		"dfid":            "-",
		"clienttime":      time.Now().UnixMilli(),
		"key":             KugouKeyAndroid,
		"fields":          "album_info,author_name,audio_info,ori_audio_name,base,songname",
		"data":            []map[string]string{{"hash": hash}},
	}
	var resp struct {
		ErrorCode FInt            `json:"error_code"`
		Data      json.RawMessage `json:"data"`
	}
	if err := k.C.PostJSON(ctx, kugouGateway, map[string]string{
		"KG-THash":   "13a3164",
		"KG-RC":      "1",
		"KG-Fake":    "0",
		"KG-RF":      "00869891",
		"x-router":   "kmr.service.kugou.com",
		"User-Agent": "Android712-AndroidPhone-11451-376-0-FeeCacheUpdate-wifi",
	}, body, &resp); err != nil {
		return err
	}
	if resp.ErrorCode != 0 {
		return fmt.Errorf("kg: gateway error_code=%d for %s", resp.ErrorCode, s.KeyString())
	}
	entries, err := kugouEntries(resp.Data)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("kg: gateway returned no data for %s", s.KeyString())
	}
	e := entries[0]
	ai := e.AudioInfo
	if s.Name == "" {
		s.Name = string(e.SongName)
	}
	if s.Singer == "" {
		s.Singer = string(e.AuthorName)
	}
	if s.Album == "" {
		s.Album = string(e.AlbumInfo.AlbumName)
	}
	if s.AlbumID == "" {
		s.AlbumID = string(e.AlbumInfo.AlbumID)
	}
	merge := func(t string, hash, size FStr) {
		if hash == "" || contains(qualityTypes(s.Qualities), t) {
			return
		}
		s.Qualities = append(s.Qualities, Quality{
			Type: t, Size: parseInt64(string(size)), Hash: string(hash),
		})
	}
	merge(Q128, ai.Hash128, ai.Filesize128)
	merge(Q320, ai.Hash320, ai.Filesize320)
	merge(QFlac, ai.HashFlac, ai.FilesizeFlac)
	merge(QFlac24, ai.HashHigh, ai.FilesizeHigh)
	return nil
}

// kugouEntry is one row of the gateway's rendition matrix.
type kugouEntry struct {
	AudioInfo struct {
		Hash         FStr `json:"hash"`
		Filesize     FStr `json:"filesize"`
		Hash128      FStr `json:"hash_128"`
		Filesize128  FStr `json:"filesize_128"`
		Hash320      FStr `json:"hash_320"`
		Filesize320  FStr `json:"filesize_320"`
		HashFlac     FStr `json:"hash_flac"`
		FilesizeFlac FStr `json:"filesize_flac"`
		HashHigh     FStr `json:"hash_high"`
		FilesizeHigh FStr `json:"filesize_high"`
	} `json:"audio_info"`
	AuthorName FStr `json:"author_name"`
	SongName   FStr `json:"songname"`
	AlbumInfo  struct {
		AlbumID   FStr `json:"album_id"`
		AlbumName FStr `json:"album_name"`
	} `json:"album_info"`
}

// kugouEntries decodes the gateway's data field, which is not a stable shape.
// A hash the server knows comes back as [[{...}]], and one it does not know as
// [{}] - a one-level array holding a single empty object. That reply is the
// server's way of saying "nothing here", so both shapes are tried in turn and
// rows without an audio hash or a song name are dropped: a wrong or truncated
// hash has to read as "no data", not as a Go unmarshal error in the output.
// Only a reply that is unreadable in either shape is an error.
func kugouEntries(raw json.RawMessage) ([]kugouEntry, error) {
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 || string(trim) == "null" {
		return nil, nil
	}
	var entries []kugouEntry
	if err := json.Unmarshal(trim, &entries); err != nil {
		var nested [][]kugouEntry
		if err := json.Unmarshal(trim, &nested); err != nil {
			return nil, err
		}
		for _, row := range nested {
			entries = append(entries, row...)
		}
	}
	for i, e := range entries {
		if e.AudioInfo.Hash != "" || e.SongName != "" {
			return entries[i:], nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------- url / pic / lyric

// privilegeBody is the get_res_privilege payload shared by MusicURL and Pic.
func (k *Kugou) privilegeBody(s *Song, hash string) map[string]any {
	audioID := s.MID
	if audioID == "" {
		audioID = strings.TrimSuffix(s.ID, s.Hash)
	}
	name := s.Name
	if s.Singer != "" {
		name = s.Singer + " - " + name
	}
	return map[string]any{
		"appid":            1001,
		"area_code":        "1",
		"behavior":         "play",
		"clientver":        "9020",
		"need_hash_offset": 1,
		"relate":           1,
		"resource": []map[string]any{{
			"album_audio_id": audioID,
			"album_id":       s.AlbumID,
			"hash":           hash,
			"id":             0,
			"name":           name + ".mp3",
			"type":           "audio",
		}},
		"token": "",
		// Pinned to int64: 2626431536 exceeds int on 32-bit targets, and an
		// untyped literal stored in an any silently narrows to int.
		"userid": int64(2626431536),
		"vip":    1,
	}
}

type kugouPrivResp struct {
	ErrorCode FInt `json:"error_code"`
	Message   FStr `json:"message"`
	Data      []struct {
		Hash      FStr `json:"hash"`
		Privilege FInt `json:"privilege"`
		PayType   FInt `json:"pay_type"`
		Price     FInt `json:"price"`
		Info      struct {
			DownloadURL FStr      `json:"downloadurl"`
			Image       FStr      `json:"image"`
			ImgSize     FStrSlice `json:"imgsize"`
		} `json:"info"`
	} `json:"data"`
}

// MusicURL reports why kugou will not hand over audio.
//
// get_res_privilege is the only kugou endpoint the desktop app calls for audio,
// and it never returns a url: its info object carries duration, filesize,
// bitrate, extname and the cover image, and nothing else. Checked live on
// 2026-10-10 for a paywalled track (privilege 10, pay_type 3, price 200) and a
// free one (privilege 0, pay_type 0, price 0) - both replies omitted
// info.downloadurl entirely, and the free one proves the omission is the shape
// of the answer rather than a block on this particular song. renderer.js shows
// the division of labour: the app calls the same endpoint only to read
// info.image for the cover, and its kugou source object has no downloadUrl
// method at all. Audio is delegated to the User API, exactly like wy and tx.
func (k *Kugou) MusicURL(ctx context.Context, s *Song, quality Quality) (*PlayerURL, error) {
	hash := quality.Hash
	if hash == "" {
		hash = s.Hash
	}
	if hash == "" {
		return nil, fmt.Errorf("kg: no file hash for %s", s.KeyString())
	}
	var resp kugouPrivResp
	if err := k.C.PostJSON(ctx, kugouPrivilege, kugouHeaders, k.privilegeBody(s, hash), &resp); err != nil {
		return nil, err
	}
	if resp.ErrorCode != 0 {
		return nil, fmt.Errorf("kg: get_res_privilege error_code=%d: %s",
			resp.ErrorCode, strings.TrimSpace(string(resp.Message)))
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("kg: empty response for %s", s.KeyString())
	}
	d := resp.Data[0]
	if pic := k.coverURL(string(d.Info.Image), d.Info.ImgSize); pic != "" {
		s.Pic = pic
	}
	down := string(d.Info.DownloadURL)
	if down != "" {
		return &PlayerURL{
			URL: down, Quality: quality.Type, Source: k.ID(),
			Size: quality.Size, Duration: s.Duration, FileName: s.Name + ".mp3",
			Headers: map[string]string{
				"User-Agent": "KuGou2012-9020-ExpandSearchManager",
				"Referer":    "https://www.kugou.com/",
			},
		}, nil
	}
	// The request reached the server and the reply is well formed, so an empty
	// url is an answer and not a failure. Report the one the reply actually
	// gives instead of guessing at VIP locks and blocked regions. No "kg:"
	// source tag here: s.KeyString() already carries it, and prefixing it
	// produced the adjacent "kg: kg:<hash>" in the picker's status line.
	if d.Privilege != 0 || d.PayType != 0 {
		return nil, fmt.Errorf("%s is a paid track (privilege=%d pay_type=%d price=%d): kugou gives no anonymous audio url, so add a user source that serves kg or use a paid account",
			s.KeyString(), d.Privilege, d.PayType, d.Price)
	}
	return nil, fmt.Errorf("%s: kugou gives no anonymous audio url (privilege=%d pay_type=%d) - get_res_privilege returns privilege and cover data only. add a user source that serves kg audio, e.g. lxmusic source add --name hyw --url <script.js>",
		s.KeyString(), d.Privilege, d.PayType)
}

// coverURL fills the "{size}" placeholder exactly like the app does.
func (k *Kugou) coverURL(image string, imgsize []string) string {
	if image == "" {
		return ""
	}
	if i := strings.Index(image, "{size}"); i >= 0 {
		size := "1920"
		if len(imgsize) > 0 {
			size = imgsize[0]
		}
		return image[:i] + size + image[i+len("{size}"):]
	}
	return image
}

// Pic returns the cover URL, resolving it through get_res_privilege if needed.
func (k *Kugou) Pic(ctx context.Context, s *Song) (string, error) {
	if s.Pic != "" {
		return s.Pic, nil
	}
	hash := s.Hash
	if hash == "" {
		return "", fmt.Errorf("kg: no hash for %s", s.KeyString())
	}
	var resp kugouPrivResp
	if err := k.C.PostJSON(ctx, kugouPrivilege, kugouHeaders, k.privilegeBody(s, hash), &resp); err != nil {
		return "", err
	}
	if resp.ErrorCode != 0 {
		return "", fmt.Errorf("kg: get_res_privilege error_code=%d", resp.ErrorCode)
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("kg: no cover for %s", s.KeyString())
	}
	pic := k.coverURL(string(resp.Data[0].Info.Image), resp.Data[0].Info.ImgSize)
	if pic == "" {
		return "", fmt.Errorf("kg: empty cover for %s", s.KeyString())
	}
	s.Pic = pic
	return pic, nil
}

// Lyric searches the kugou lyric service by name + hash, then downloads it.
// LRC payloads are base64 text; KRC payloads are XOR-encrypted and inflated
// into tagged lines plus a language block (see lyric.go for the decoder).
func (k *Kugou) Lyric(ctx context.Context, s *Song) (*Lyric, error) {
	hash := s.Hash
	if hash == "" {
		return nil, fmt.Errorf("kg: no hash for %s", s.KeyString())
	}
	keyword := s.Name
	if s.Singer != "" {
		keyword = s.Singer + " " + s.Name
	}
	q := url.Values{}
	q.Set("ver", "1")
	q.Set("man", "yes")
	q.Set("client", "pc")
	q.Set("keyword", keyword)
	q.Set("hash", hash)
	q.Set("timelength", strconv.Itoa(s.Duration))
	q.Set("lrctxt", "1")
	var resp struct {
		Candidates []struct {
			ID          FStr `json:"id"`
			AccessKey   FStr `json:"accesskey"`
			KrcType     FInt `json:"krctype"`
			ContentType FInt `json:"contenttype"`
		} `json:"candidates"`
	}
	if err := k.C.GetJSON(ctx, kugouLyricFind+"?"+q.Encode(), kugouHeaders, &resp); err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("kg: lyric not found for %s", s.KeyString())
	}
	c := resp.Candidates[0]
	kind := "lrc"
	if c.KrcType == 1 && c.ContentType != 1 {
		kind = "krc"
	}
	dq := url.Values{}
	dq.Set("ver", "1")
	dq.Set("client", "pc")
	dq.Set("id", string(c.ID))
	dq.Set("accesskey", string(c.AccessKey))
	dq.Set("fmt", kind)
	dq.Set("charset", "utf8")
	var got struct {
		Format  FStr `json:"fmt"`
		Content FStr `json:"content"`
	}
	if err := k.C.GetJSON(ctx, kugouLyricDL+"?"+dq.Encode(), kugouHeaders, &got); err != nil {
		return nil, err
	}
	l := DecodeKugouLyric(string(got.Content), kind == "krc")
	l.URL = kugouLyricDL
	return l, nil
}

// ------------------------------------------------------------- helpers

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func contains(list []string, v string) bool {
	if v == "" {
		return false
	}
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
