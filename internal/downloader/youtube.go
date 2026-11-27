/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package downloader

import (
	"istkharmusic/internal/config"
	"istkharmusic/internal/utils"
	"time"

	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// youTubeData provides an interface for fetching track and playlist information from YouTube.
type youTubeData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

var youtubePatterns = map[string]*regexp.Regexp{
	"youtube":   regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/.*`),
	"youtu_be":  regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtu\.be/.*`),
	"yt_music":  regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/.*`),
	"yt_shorts": regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/shorts/.*`),
}

// newYouTubeData initializes a youTubeData instance with pre-compiled regex patterns and a cleaned query.
func newYouTubeData(query string) *youTubeData {
	return &youTubeData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.ApiUrl, "/"),
		APIKey:   config.ApiKey,
		Patterns: youtubePatterns,
	}
}

func (y *youTubeData) isValid() bool {
	if y.Query == "" {
		slog.Info("The query or patterns are empty.")
		return false
	}

	for _, pattern := range y.Patterns {
		if pattern.MatchString(y.Query) {
			return true
		}
	}
	return false
}

func (y *youTubeData) getInfo() (*utils.PlatformTracks, error) {
	if !y.isValid() {
		return nil, errors.New("the provided URL is invalid or the platform is not supported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	y.Query = normalizeYouTubeURL(y.Query)
	videoID := extractVideoID(y.Query)
	playlistID := extractPlaylistID(y.Query)

	switch {
	case playlistID != "":
		var res *utils.PlatformTracks
		var perr error
		if strings.HasPrefix(playlistID, "RD") && !strings.HasPrefix(playlistID, "RDCLAK") {
			res, perr = GetYouTubeMixPlaylist(ctx, playlistID)
		} else {
			res, perr = getYouTubePlaylist(ctx, playlistID)
		}
		if (perr != nil || res == nil || len(res.Results) == 0) && riteshReady() {
			if tr, err := riteshPlaylist(y.Query, 25); err == nil {
				return &utils.PlatformTracks{Results: tr}, nil
			}
		}
		return res, perr

	case videoID != "":
		for _, query := range []string{videoID, y.Query} {
			tracks, err := searchYouTube(query, 10)
			if err != nil {
				continue
			}

			for _, track := range tracks {
				if track.Id == videoID {
					return &utils.PlatformTracks{Results: []utils.GetUrlTrack{track}}, nil
				}
			}
		}

		if title, err := getYouTubeTitleFromOEmbed(videoID); err == nil && title != "" {
			tracks, err := searchYouTube(title, 10)
			if err == nil {
				for _, track := range tracks {
					if track.Id == videoID {
						return &utils.PlatformTracks{Results: []utils.GetUrlTrack{track}}, nil
					}
				}
			}
		}

		slog.Warn("Video ID was extracted but no matching track was found in search results", "video_id", videoID)
		if riteshReady() {
			if t, err := riteshDetails(y.Query); err == nil {
				return &utils.PlatformTracks{Results: []utils.GetUrlTrack{*t}}, nil
			}
		}
		return getYouTubeVideo(ctx, videoID)
	}

	return nil, errors.New("no video or playlist results were found")
}

func (y *youTubeData) search() (*utils.PlatformTracks, error) {
	if y.isValid() {
		return y.getInfo()
	}

	tracks, err := searchYouTube(y.Query, 5)
	if (err != nil || len(tracks) == 0) && riteshReady() {
		if rt, rerr := riteshSearch(y.Query, 5); rerr == nil {
			return &utils.PlatformTracks{Results: rt}, nil
		}
	}
	if err != nil {
		return nil, err
	}

	return &utils.PlatformTracks{Results: tracks}, nil
}

func (y *youTubeData) getTrack() (*utils.TrackInfo, error) {
	if y.Query == "" {
		return nil, errors.New("the query is empty")
	}

	if !y.isValid() {
		return nil, errors.New("the provided URL is invalid or the platform is not supported")
	}

	getInfo, err := y.getInfo()
	if err != nil {
		return nil, err
	}
	if len(getInfo.Results) == 0 {
		return nil, errors.New("no video results were found")
	}

	track := getInfo.Results[0]
	trackInfo := &utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		Platform: utils.YouTube,
	}

	return trackInfo, nil
}

// downloadTrack downloads from the Ritesh API first, then falls back to yt-dlp.
func (y *youTubeData) downloadTrack(info *utils.TrackInfo, video bool) (string, error) {
	if riteshReady() {
		path, err := riteshDownloadID(info.Id, video)
		if err == nil {
			return path, nil
		}
		slog.Warn("[youtube] Ritesh API download failed, falling back to yt-dlp", "error", redact(err))
	}
	return y.downloadWithYtDlp(info.Id, video)
}

// buildYtdlpParams constructs the command-line parameters for yt-dlp to download media.
func (y *youTubeData) buildYtdlpParams(videoID string, video bool) ([]string, string) {
	outputTemplate := filepath.Join(config.DownloadsDir, "%(id)s.%(ext)s")
	params := []string{
		"yt-dlp",
		"--no-warnings",
		"--quiet",
		"--geo-bypass",
		"--retries", "2",
		"--continue",
		"--no-part",
		"--concurrent-fragments", "3",
		"--socket-timeout", "10",
		"--throttled-rate", "100K",
		"--retry-sleep", "1",
		"--no-write-thumbnail",
		"--no-write-info-json",
		"--no-embed-metadata",
		"--no-embed-chapters",
		"--no-embed-subs",
		"--extractor-args", "youtube:player_js_version=actual",
		"-o", outputTemplate,
	}

	if video {
		formatSelector := "bestvideo[height<=720]+bestaudio/best[height<=720]"
		params = append(params, "-f", formatSelector, "--merge-output-format", "mp4")
	} else {
		params = append(params, "-f", "bestaudio[ext=m4a]/bestaudio")
	}

	if config.Proxy != "" {
		params = append(params, "--proxy", config.Proxy)
	}

	videoURL := "https://www.youtube.com/watch?v=" + videoID
	params = append(params, videoURL, "--print", "after_move:filepath")

	return params, ""
}

// downloadWithYtDlp downloads media from YouTube using the yt-dlp command-line tool.
func (y *youTubeData) downloadWithYtDlp(videoID string, video bool) (string, error) {
	if videoID == "" {
		return "", errors.New("videoID is empty")
	}

	ytdlpParams, _ := y.buildYtdlpParams(videoID, video)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytdlpParams[0], ytdlpParams[1:]...)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			stderr := string(exitErr.Stderr)
			return "", fmt.Errorf("yt-dlp failed with exit code %d: %s", exitErr.ExitCode(), stderr)
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("yt-dlp timed out for video ID: %s", videoID)
		}

		return "", fmt.Errorf("an unexpected error occurred while downloading %s: %w", videoID, err)
	}

	downloadedPathStr := strings.TrimSpace(string(output))
	if downloadedPathStr == "" {
		return "", fmt.Errorf("no output path was returned for %s", videoID)
	}

	if _, err := os.Stat(downloadedPathStr); os.IsNotExist(err) {
		return "", fmt.Errorf("the file was not found at the reported path: %s", downloadedPathStr)
	}

	return downloadedPathStr, nil
}


// getRecommendations retrieves native YouTube mix recommendations without external API.
func (y *youTubeData) getRecommendations(ctx context.Context, rawURL string) (*utils.PlatformTracks, error) {
	urlToUse := rawURL
	if urlToUse == "" {
		urlToUse = y.Query
	}

	videoID := extractVideoID(urlToUse)
	if videoID == "" && urlToUse != "" && !strings.HasPrefix(urlToUse, "http") {
		if len(urlToUse) == 11 {
			videoID = urlToUse
		}
	}

	if videoID == "" {
		return nil, errors.New("no valid YouTube video ID found for recommendations")
	}

	return GetYouTubeMixPlaylist(ctx, "RD"+videoID)
}
