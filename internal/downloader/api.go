/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package downloader

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"istkharmusic/internal/utils"
)

// apiData resolves Spotify links and 90+ other platforms through the Ritesh API.
type apiData struct {
	Query string
}

var spotifyRe = regexp.MustCompile(`(?i)open\.spotify\.com/(?:intl-[a-z-]+/)?(?:embed/)?(track|album|playlist|artist)/([A-Za-z0-9]{22})`)

func newApiData(query string) *apiData {
	return &apiData{Query: strings.TrimSpace(query)}
}

func (a *apiData) isSpotify() bool { return spotifyRe.MatchString(a.Query) }

func (a *apiData) isValid() bool {
	if a.Query == "" || !riteshReady() {
		return false
	}
	return a.isSpotify() || isSupportedPlatformURL(a.Query)
}

func (a *apiData) getInfo() (*utils.PlatformTracks, error) {
	if !a.isValid() {
		return nil, errors.New("the provided URL is invalid or the platform is not supported")
	}

	if m := spotifyRe.FindStringSubmatch(a.Query); m != nil {
		if strings.EqualFold(m[1], "track") {
			t, err := riteshDetails(a.Query)
			if err != nil {
				return nil, err
			}
			return &utils.PlatformTracks{Results: []utils.GetUrlTrack{*t}}, nil
		}
		tracks, err := riteshPlaylist(a.Query, 25)
		if err != nil {
			return nil, err
		}
		return &utils.PlatformTracks{Results: tracks}, nil
	}

	ex, err := riteshExtractMedia(a.Query)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(ex.Title)
	if title == "" {
		title = "Media"
		if ex.Platform != "" {
			title = strings.ToUpper(ex.Platform[:1]) + ex.Platform[1:] + " Media"
		}
	}
	return &utils.PlatformTracks{Results: []utils.GetUrlTrack{{
		Title: title, Id: ex.ID, Url: a.Query, Thumbnail: ex.Thumbnail,
		Duration: ex.Duration, Platform: PlatformRitesh,
	}}}, nil
}

func (a *apiData) search() (*utils.PlatformTracks, error) {
	if a.isValid() {
		return a.getInfo()
	}
	tracks, err := riteshSearch(a.Query, 5)
	if err != nil {
		return nil, err
	}
	return &utils.PlatformTracks{Results: tracks}, nil
}

func (a *apiData) getTrack() (*utils.TrackInfo, error) {
	if !riteshReady() {
		return nil, errors.New("API_URL / API_KEY is not configured")
	}
	// Spotify → matched YouTube track.
	if a.isSpotify() {
		t, err := riteshDetails(a.Query)
		if err != nil {
			return nil, err
		}
		return &utils.TrackInfo{Id: t.Id, URL: t.Url, Platform: utils.YouTube}, nil
	}
	if isSupportedPlatformURL(a.Query) {
		return &utils.TrackInfo{Id: safeName(a.Query), URL: a.Query, Platform: PlatformRitesh}, nil
	}
	return nil, errors.New("the provided URL is not supported")
}

func (a *apiData) downloadTrack(info *utils.TrackInfo, video bool) (string, error) {
	if info.Platform == utils.YouTube {
		return newYouTubeData(info.URL).downloadTrack(info, video)
	}
	return riteshDownloadExtracted(info.URL, video)
}

// Ritesh has no recommendations endpoint; autoplay uses YouTube mixes instead.
func (a *apiData) getRecommendations(_ context.Context, _ string) (*utils.PlatformTracks, error) {
	return nil, errors.New("recommendations are not available for this platform")
}
