/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package downloader

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"istkharmusic/internal/config"
	"istkharmusic/internal/utils"

	"context"
)

// Ritesh Music API client (https://www.riteshyt.in/docs).
// The API key is always sent as the `api_key` query parameter.

const (
	riteshAudioTimeout = 300 * time.Second
	riteshVideoTimeout = 600 * time.Second
	riteshMinFileBytes = 10_000
	// PlatformRitesh marks tracks resolved through /api/platform/extract.
	PlatformRitesh = "riteshapi"
)

var riteshDlClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 4 * time.Minute,
		DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
	},
	CheckRedirect: func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return nil
	},
}

var supportedPlatformDomains = []string{
	"instagram.com", "instagr.am", "tiktok.com", "twitter.com", "x.com", "facebook.com", "fb.watch", "fb.com",
	"threads.net", "snapchat.com", "reddit.com", "v.redd.it", "pinterest.com", "pin.it", "linkedin.com", "bsky.app",
	"tumblr.com", "douyin.com", "kuaishou.com", "triller.co", "likee.video", "likee.com", "weibo.com", "weibo.cn",
	"xiaohongshu.com", "9gag.com",
	"jiosaavn.com", "saavn.com", "soundcloud.com", "music.apple.com", "apple.co", "gaana.com", "deezer.com",
	"deezer.page.link", "resso.com", "tidal.com", "bandcamp.com", "mixcloud.com", "audiomack.com", "hearthis.at",
	"jamendo.com", "reverbnation.com", "audius.co", "boomplay.com", "anghami.com", "wynk.in", "hungama.com",
	"napster.com", "kugou.com", "kuwo.cn", "music.163.com", "iheart.com", "tunein.com", "pandora.com", "last.fm",
	"genius.com",
	"dailymotion.com", "dai.ly", "vimeo.com", "vk.com", "rumble.com", "streamable.com", "ok.ru", "coub.com",
	"nicovideo.jp", "odysee.com", "bitchute.com", "veoh.com", "rutube.ru", "ted.com", "archive.org", "youku.com",
	"iqiyi.com", "afreecatv.com", "trovo.live", "twitch.tv", "kick.com", "bilibili.com", "b23.tv", "mxplayer.in",
	"redbull.com", "newgrounds.com", "imdb.com",
	"imgur.com", "loom.com", "wistia.com", "flickr.com", "giphy.com", "tenor.com",
	"terabox.com", "teraboxapp.com", "1024tera.com",
}

func isSupportedPlatformURL(query string) bool {
	q := strings.TrimSpace(query)
	if !strings.HasPrefix(strings.ToLower(q), "http://") && !strings.HasPrefix(strings.ToLower(q), "https://") {
		return false
	}
	u, err := url.Parse(q)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, "youtube.com") || host == "youtu.be" {
		return false
	}
	for _, d := range supportedPlatformDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func riteshReady() bool {
	return strings.TrimSpace(config.ApiUrl) != "" && strings.TrimSpace(config.ApiKey) != ""
}

func riteshBase() string { return strings.TrimRight(config.ApiUrl, "/") }

// redact removes the API key from error text so it never reaches logs or chats.
func redact(err error) error {
	if err == nil || config.ApiKey == "" {
		return err
	}
	msg := strings.ReplaceAll(err.Error(), config.ApiKey, "***")
	msg = strings.ReplaceAll(msg, url.QueryEscape(config.ApiKey), "***")
	return errors.New(msg)
}

func riteshGet(path string, q url.Values, out any) error {
	if !riteshReady() {
		return errors.New("ritesh api is not configured (API_URL / API_KEY)")
	}
	q.Set("api_key", config.ApiKey)
	resp, err := sendRequest(http.MethodGet, riteshBase()+path+"?"+q.Encode(), nil, nil)
	if err != nil {
		return redact(fmt.Errorf("ritesh %s: %w", path, err))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return redact(err)
	}
	if err = json.Unmarshal(body, out); err != nil && resp.StatusCode == http.StatusOK {
		return fmt.Errorf("ritesh %s: bad response: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ritesh %s: HTTP %d", path, resp.StatusCode)
	}
	return nil
}

func parseClock(s string) int32 {
	parts := strings.Split(strings.TrimSpace(s), ":")
	var total int
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return int32(total)
}

func ytURL(id string) string { return "https://www.youtube.com/watch?v=" + id }

// riteshSearch → GET /search?q=&limit=
func riteshSearch(query string, limit int) ([]utils.GetUrlTrack, error) {
	var resp struct {
		Result []struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			Duration   string `json:"duration"`
			Link       string `json:"link"`
			Thumbnails []struct {
				URL string `json:"url"`
			} `json:"thumbnails"`
		} `json:"result"`
	}
	if err := riteshGet("/search", url.Values{"q": {strings.TrimSpace(query)}, "limit": {strconv.Itoa(limit)}}, &resp); err != nil {
		return nil, err
	}
	var out []utils.GetUrlTrack
	for _, r := range resp.Result {
		if r.ID == "" || r.Title == "" {
			continue
		}
		thumb := ""
		if n := len(r.Thumbnails); n > 0 {
			thumb = r.Thumbnails[n-1].URL
		}
		link := r.Link
		if link == "" {
			link = ytURL(r.ID)
		}
		out = append(out, utils.GetUrlTrack{
			Title: r.Title, Id: r.ID, Url: link, Thumbnail: thumb,
			Duration: parseClock(r.Duration), Platform: utils.YouTube,
		})
	}
	if len(out) == 0 {
		return nil, errors.New("ritesh search returned no tracks")
	}
	return out, nil
}

// riteshDetails → GET /details?link= (YouTube link/id or Spotify track).
func riteshDetails(link string) (*utils.GetUrlTrack, error) {
	var r struct {
		Title       string `json:"title"`
		DurationMin string `json:"duration_min"`
		DurationSec int32  `json:"duration_sec"`
		Thumbnail   string `json:"thumbnail"`
		VidID       string `json:"vidid"`
		Link        string `json:"link"`
	}
	if err := riteshGet("/details", url.Values{"link": {strings.TrimSpace(link)}}, &r); err != nil {
		return nil, err
	}
	if r.Title == "" || r.VidID == "" {
		return nil, errors.New("ritesh details returned no data")
	}
	dur := r.DurationSec
	if dur <= 0 {
		dur = parseClock(r.DurationMin)
	}
	u := r.Link
	if u == "" {
		u = ytURL(r.VidID)
	}
	return &utils.GetUrlTrack{Title: r.Title, Id: r.VidID, Url: u, Thumbnail: r.Thumbnail, Duration: dur, Platform: utils.YouTube}, nil
}

// riteshPlaylist → GET /playlist?link=&limit= (YouTube / Spotify playlists).
func riteshPlaylist(link string, limit int) ([]utils.GetUrlTrack, error) {
	var resp struct {
		Videos []struct {
			VidID       string `json:"vidid"`
			Title       string `json:"title"`
			DurationMin string `json:"duration_min"`
			DurationSec int32  `json:"duration_sec"`
			Thumbnail   string `json:"thumbnail"`
		} `json:"videos"`
	}
	if err := riteshGet("/playlist", url.Values{"link": {strings.TrimSpace(link)}, "limit": {strconv.Itoa(limit)}}, &resp); err != nil {
		return nil, err
	}
	var out []utils.GetUrlTrack
	for _, v := range resp.Videos {
		if v.VidID == "" || v.Title == "" {
			continue
		}
		dur := v.DurationSec
		if dur <= 0 {
			dur = parseClock(v.DurationMin)
		}
		out = append(out, utils.GetUrlTrack{Title: v.Title, Id: v.VidID, Url: ytURL(v.VidID), Thumbnail: v.Thumbnail, Duration: dur, Platform: utils.YouTube})
	}
	if len(out) == 0 {
		return nil, errors.New("ritesh playlist returned no tracks")
	}
	return out, nil
}

type riteshExtract struct {
	Success     bool   `json:"success"`
	Platform    string `json:"platform"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Thumbnail   string `json:"thumbnail"`
	Duration    int32  `json:"duration"`
	MediaType   string `json:"media_type"`
	DownloadURL string `json:"download_url"`
	Error       string `json:"error"`
}

// riteshExtractMedia → GET /api/platform/extract?url= (90+ sites).
func riteshExtractMedia(link string) (*riteshExtract, error) {
	var r riteshExtract
	err := riteshGet("/api/platform/extract", url.Values{"url": {strings.TrimSpace(link)}}, &r)
	if err != nil && !r.Success {
		if r.Error != "" {
			return nil, fmt.Errorf("ritesh extract failed: %s", r.Error)
		}
		return nil, err
	}
	if !r.Success || r.DownloadURL == "" {
		msg := r.Error
		if msg == "" {
			msg = "no download url"
		}
		return nil, fmt.Errorf("ritesh extract failed: %s", msg)
	}
	if r.ID == "" {
		h := md5.Sum([]byte(link))
		r.ID = hex.EncodeToString(h[:8])
	}
	return &r, nil
}

func safeName(id string) string {
	id = sanitizeFilename(id)
	id = strings.ReplaceAll(id, " ", "_")
	if id == "" {
		id = generateUniqueName("")
	}
	if len(id) > 80 {
		id = id[:80]
	}
	return id
}

func cachedFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() >= riteshMinFileBytes
}

// fetchToFile streams rawURL into dest and validates it is real media.
func fetchToFile(rawURL, dest string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return redact(err)
	}
	req.Header.Set("User-Agent", "IstkharMusic/1.0")

	resp, err := riteshDlClient.Do(req)
	if err != nil {
		return redact(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") || strings.Contains(ct, "text/html") {
		return errors.New("download returned a non-media response")
	}

	if err = os.MkdirAll(filepath.Dir(dest), defaultDownloadDirPerm); err != nil {
		return err
	}
	tmp := dest + ".part"
	if err = writeToFile(tmp, resp.Body); err != nil {
		_ = os.Remove(tmp)
		return redact(err)
	}
	st, err := os.Stat(tmp)
	if err != nil || st.Size() < riteshMinFileBytes {
		_ = os.Remove(tmp)
		return errors.New("download returned an empty or invalid file")
	}
	return os.Rename(tmp, dest)
}

// riteshDownloadID downloads a YouTube video id via /download (+ musicbot fallback for audio).
func riteshDownloadID(id string, video bool) (string, error) {
	if !riteshReady() {
		return "", errors.New("ritesh api is not configured")
	}
	if id == "" {
		return "", errors.New("missing video id")
	}
	dtype, quality, ext, timeout := "audio", "320k", ".mp3", riteshAudioTimeout
	if video {
		dtype, quality, ext, timeout = "video", "720p", ".mp4", riteshVideoTimeout
	}
	dest := filepath.Join(config.DownloadsDir, safeName(id)+ext)
	if cachedFile(dest) {
		return dest, nil
	}

	primary := fmt.Sprintf("%s/download?%s", riteshBase(), url.Values{
		"query": {id}, "dl_type": {dtype}, "quality": {quality}, "api_key": {config.ApiKey},
	}.Encode())
	err := fetchToFile(primary, dest, timeout)
	if err == nil {
		return dest, nil
	}
	if video {
		return "", fmt.Errorf("ritesh video download: %w", err)
	}

	fallback := fmt.Sprintf("%s/api/musicbot/play/%s?%s", riteshBase(), url.PathEscape(id), url.Values{"api_key": {config.ApiKey}}.Encode())
	if err2 := fetchToFile(fallback, dest, timeout); err2 != nil {
		return "", fmt.Errorf("ritesh audio download failed: %w / %v", err, redact(err2))
	}
	return dest, nil
}

// riteshDownloadExtracted downloads media from any supported platform URL.
func riteshDownloadExtracted(link string, video bool) (string, error) {
	ex, err := riteshExtractMedia(link)
	if err != nil {
		return "", err
	}
	dl := ex.DownloadURL
	if strings.HasPrefix(dl, "/") {
		dl = riteshBase() + dl
	}
	if !strings.Contains(dl, "api_key=") && strings.HasPrefix(dl, riteshBase()) {
		sep := "?"
		if strings.Contains(dl, "?") {
			sep = "&"
		}
		dl += sep + "api_key=" + url.QueryEscape(config.ApiKey)
	}
	ext := ".mp3"
	if video || ex.MediaType == "video" {
		ext = ".mp4"
	}
	dest := filepath.Join(config.DownloadsDir, safeName(ex.ID)+ext)
	if cachedFile(dest) {
		return dest, nil
	}
	timeout := riteshAudioTimeout
	if ext == ".mp4" {
		timeout = riteshVideoTimeout
	}
	if err = fetchToFile(dl, dest, timeout); err != nil {
		return "", err
	}
	return dest, nil
}
