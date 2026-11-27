/*
 * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 * Copyright (c) 2025-2026 Ashok Shau
 *
 * Licensed under GNU GPL v3
 * See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package bot

import (
	"fmt"
	"html"
	"istkharmusic/internal/cache"
	"istkharmusic/internal/calls"
	"istkharmusic/internal/db"
	"istkharmusic/internal/utils"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func wrapQuoteHTML(inner string) string {
	return "<blockquote>" + inner + "</blockquote>"
}

// panelEdit re-renders the now-playing panel (rich message) after a control press.
func panelEdit(c *td.Client, cb *td.UpdateNewCallbackQuery, track *utils.PlayerCache, status, extra, mode string) {
	var played uint64
	if p, err := calls.Calls.PlayedTime(cb.ChatId); err == nil {
		played = p
	}
	markup := utils.PanelButtons(mode, int32(played), track.Duration, cache.ChatCache.GetAutoplay(cb.ChatId))
	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: wrapQuoteHTML(utils.NowPlayingHTML(track, status, extra))}}
	if _, err := c.EditRichMessage(cb.ChatId, cb.MessageId, rich, &td.EditRichMessageOpts{ReplyMarkup: markup}); err != nil {
		plain := fmt.Sprintf("<blockquote><b>%s</b>\n\n<b>%s:</b> <a href='%s'>%s</a>\n<b>%s:</b> %s\n<b>%s:</b> %s",
			status, utils.SC("track"), html.EscapeString(track.URL), html.EscapeString(track.Name),
			utils.SC("duration"), utils.SecToMin(track.Duration), utils.SC("requester"), html.EscapeString(track.User))
		if extra != "" {
			plain += "\n" + html.EscapeString(extra)
		}
		plain += "</blockquote>"
		_, _ = cb.EditMessageText(c, plain, &td.EditMessageTextOpts{ReplyMarkup: markup, ParseMode: "HTML", DisableWebPagePreview: true})
	}
}

// panelNote replaces the panel with a short note (close button only).
func panelNote(c *td.Client, cb *td.UpdateNewCallbackQuery, text string) {
	body := wrapQuoteHTML("<b>" + html.EscapeString(text) + "</b>")
	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: "<p>" + body + "</p>"}}
	if _, err := c.EditRichMessage(cb.ChatId, cb.MessageId, rich, &td.EditRichMessageOpts{ReplyMarkup: utils.ControlButtons("")}); err != nil {
		_, _ = cb.EditMessageText(c, body, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
	}
}

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()
	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}
	by := utils.SC("by") + " " + user.FirstName

	if !cache.ChatCache.IsActive(chatID) {
		_ = cb.Answer(c, utils.SC("there is no active playback."), nil)
		panelNote(c, cb, utils.SC("there is no active playback."))
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, utils.SC("there is no active playback."), nil)
		panelNote(c, cb, utils.SC("there is no active playback."))
		return nil
	}

	switch {
	case strings.Contains(data, "play_progress"):
		played, _ := calls.Calls.PlayedTime(chatID)
		_ = cb.Answer(c, fmt.Sprintf("%s / %s", utils.SecToMin(int32(played)), utils.SecToMin(currentTrack.Duration)), nil)
		panelEdit(c, cb, currentTrack, utils.SC("now playing"), "", "play")
		return nil

	case strings.Contains(data, "play_replay"), strings.Contains(data, "play_seek_"):
		var delta int
		switch {
		case strings.Contains(data, "play_seek_back"):
			delta = -15
		case strings.Contains(data, "play_seek_fwd"):
			delta = 15
		default:
			played, _ := calls.Calls.PlayedTime(chatID)
			delta = -int(played)
		}
		if err := calls.Calls.SeekStream(c, chatID, delta); err != nil {
			_ = cb.Answer(c, utils.SC("unable to seek")+": "+err.Error(), nil)
			return nil
		}
		switch {
		case delta == 15:
			_ = cb.Answer(c, utils.SC("forwarded 15s."), nil)
		case delta == -15:
			_ = cb.Answer(c, utils.SC("rewound 15s."), nil)
		default:
			_ = cb.Answer(c, utils.SC("replaying track."), nil)
		}
		panelEdit(c, cb, currentTrack, utils.SC("now playing"), by, "play")
		return nil

	case strings.Contains(data, "play_autoplay"):
		state := !cache.ChatCache.GetAutoplay(chatID)
		cache.ChatCache.SetAutoplay(chatID, state)
		if state {
			_ = cb.Answer(c, utils.SC("autoplay enabled."), nil)
		} else {
			_ = cb.Answer(c, utils.SC("autoplay disabled."), nil)
		}
		panelEdit(c, cb, currentTrack, utils.SC("now playing"), "", "play")
		return nil

	case strings.Contains(data, "play_skip"):
		if err := calls.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, utils.SC("unable to skip the current track."), nil)
			panelNote(c, cb, utils.SC("unable to skip the current track."))
			return nil
		}
		_ = cb.Answer(c, utils.SC("track skipped."), nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		if err := calls.Calls.Stop(chatID, false); err != nil {
			_ = cb.Answer(c, utils.SC("unable to stop playback."), nil)
			panelNote(c, cb, utils.SC("unable to stop playback."))
			return nil
		}
		_ = cb.Answer(c, utils.SC("playback stopped."), nil)
		panelNote(c, cb, utils.SC("playback stopped")+" • "+by)
		return nil

	case strings.Contains(data, "play_pause"):
		if _, err = calls.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, utils.SC("unable to pause playback."), nil)
			return nil
		}
		_ = cb.Answer(c, utils.SC("playback paused."), nil)
		panelEdit(c, cb, currentTrack, utils.SC("paused"), by, "pause")
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := calls.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, utils.SC("unable to resume playback."), nil)
			return nil
		}
		_ = cb.Answer(c, utils.SC("playback resumed."), nil)
		panelEdit(c, cb, currentTrack, utils.SC("now playing"), by, "play")
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := calls.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, utils.SC("unable to mute playback."), nil)
			return nil
		}
		_ = cb.Answer(c, utils.SC("playback muted."), nil)
		panelEdit(c, cb, currentTrack, utils.SC("muted"), by, "play")
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := calls.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, utils.SC("unable to unmute playback."), nil)
			return nil
		}
		_ = cb.Answer(c, utils.SC("playback unmuted."), nil)
		panelEdit(c, cb, currentTrack, utils.SC("now playing"), by, "play")
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, utils.SC("unable to fetch playlists."), nil)
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (IstkharMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, utils.SC("unable to create playlist."), nil)
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}

		err = db.Instance.AddSongToPlaylist(playlistID, song)
		if err != nil {
			_ = cb.Answer(c, utils.SC("unable to add track to playlist."), nil)
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, utils.SC("playlist not found."), nil)
			return nil
		}

		_ = cb.Answer(c, fmt.Sprintf("%s \"%s\" %s \"%s\".", utils.SC("track"), song.Name, utils.SC("added to playlist"), playlist.Name), nil)
		return nil

	case strings.HasPrefix(data, "play_now_"):
		trackID := strings.TrimPrefix(data, "play_now_")
		if strings.HasPrefix(data, "play_now_hash_") {
			trackID = ""
			for _, track := range cache.ChatCache.GetQueue(chatID) {
				if track != nil && utils.PlayNowCallbackData(track.TrackID) == data {
					trackID = track.TrackID
					break
				}
			}
		}
		if ok := cache.ChatCache.MoveTrackToFront(chatID, trackID); !ok {
			_ = cb.Answer(c, utils.SC("track not found in queue."), nil)
			return nil
		}

		_ = cb.Answer(c, utils.SC("playing now."), nil)
		if err = calls.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, utils.SC("unable to play the track."), nil)
			return nil
		}

		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	panelEdit(c, cb, currentTrack, utils.SC("now playing"), "", "play")
	return nil
}

func vcPlayHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	if strings.Contains(data, "vcplay_close") {
		_ = cb.Answer(c, utils.SC("closing panel."), nil)
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	c.Logger.Info("Received vcplay callback", "data", data)
	return nil
}
