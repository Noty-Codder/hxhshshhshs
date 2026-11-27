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
	"istkharmusic/internal/config"
	"istkharmusic/internal/db"
	"istkharmusic/internal/downloader"
	"istkharmusic/internal/messaging"
	"istkharmusic/internal/utils"
	"slices"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func bq(s string) string {
	return "<blockquote>" + s + "</blockquote>"
}

// playHandler handles the /play command.
func playHandler(c *td.Client, m *td.Message) error {
	if !playMode(c, m) {
		return td.EndGroups
	}

	return handlePlay(c, m, false, false)
}

// vPlayHandler handles the /vplay command.
func vPlayHandler(c *td.Client, m *td.Message) error {
	if !playMode(c, m) {
		return td.EndGroups
	}

	if !config.EnableVideoPlayback {
		msg := bq("🎥 <b>" + utils.SC("video playback is currently disabled.") + "</b>\n\n" +
			utils.SC("as more people use the bot, video streaming can sometimes cause lag and reduce music quality in voice chats. to ensure a smooth listening experience for everyone, this feature has been turned off for now.") + "\n\n" +
			utils.SC("thanks for your support and understanding") + " ❤️")
		_, _ = m.ReplyText(c, msg, &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	return handlePlay(c, m, true, false)
}

// fPlayHandler handles the /fplay command.
func fPlayHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	return handlePlay(c, m, false, true)
}

// fVPlayHandler handles the /fvplay command.
func fVPlayHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if !config.EnableVideoPlayback {
		msg := bq("🎥 <b>" + utils.SC("video playback is currently disabled.") + "</b>\n\n" +
			utils.SC("as more people use the bot, video streaming can sometimes cause lag and reduce music quality in voice chats. to ensure a smooth listening experience for everyone, this feature has been turned off for now.") + "\n\n" +
			utils.SC("thanks for your support and understanding") + " ❤️")
		_, _ = m.ReplyText(c, msg, &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	return handlePlay(c, m, true, true)
}

func handlePlay(c *td.Client, m *td.Message, isVideo bool, force bool) error {
	chatID := m.ChatId

	if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen > 10 {
		_, _ = m.ReplyText(c, bq(utils.SC("queue is full (max 10 tracks). use /end to clear.")), &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	isReply := m.ReplyToMessageID() != 0
	args := Args(m)
	rMsg := m
	var err error
	if isReply {
		rMsg, err = getRepliedMessage(c, m)
		if err != nil {
			_, err = m.ReplyText(c, bq(utils.SC("invalid reply message.")), &td.SendMessageOpts{ParseMode: "HTML"})
			return err
		}
	}
	url := getUrl(rMsg)
	if isReply && args == "" && url == "" {
		args = rMsg.Text()
	}

	input := coalesce(url, args)

	if strings.HasPrefix(input, "tgpl_") {
		playlist, err := db.Instance.GetPlaylist(input)
		if err != nil {
			_, err = m.ReplyText(c, bq("❌ "+utils.SC("playlist not found.")), &td.SendMessageOpts{ParseMode: "HTML"})
			return err
		}

		tracks := db.ConvertSongsToTracks(playlist.Songs)
		if len(tracks) == 0 {
			_, err = m.ReplyText(c, bq("❌ "+utils.SC("playlist is empty.")), &td.SendMessageOpts{ParseMode: "HTML"})
			return err
		}

		updater, err := m.ReplyText(c, bq("🔍 "+utils.SC("searching playlist...")), &td.SendMessageOpts{ParseMode: "HTML"})
		if err != nil {
			c.Logger.Warn("failed to send message", "error", err)
			return td.EndGroups
		}

		return handleMultipleTracks(c, m, updater, tracks, chatID, isVideo, force)
	}

	if match := utils.TelegramMessageRegex.FindStringSubmatch(input); match != nil {
		rMsg, err = utils.GetMessage(c, input)
		if err != nil {
			c.Logger.Warn("failed to parse message", "error", err.Error())
			_, err = m.ReplyText(c, bq(utils.SC("invalid telegram link.")), &td.SendMessageOpts{ParseMode: "HTML"})
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		if !isVideo && !force && !isReply && !m.IsPrivate() && m.SenderID() > 0 && !cache.ChatCache.IsActive(chatID) {
			if _, adminErr := cache.GetUserAdmin(c, chatID, m.SenderID(), false); adminErr == nil {
				tracks := getProfileTracks(c, m.SenderID())
				if len(tracks) > 0 && !cache.ChatCache.IsActive(chatID) {
					updater, sendErr := m.ReplyText(c, bq("🔍 "+utils.SC("loading profile songs...")), &td.SendMessageOpts{ParseMode: "HTML"})
					if sendErr != nil {
						return sendErr
					}
					return handleMultipleTracks(c, m, updater, tracks, chatID, false, false)
				}
			}
		}
		usageMsg := bq("<b>" + utils.SC("usage:") + "</b>\n/play [" + utils.SC("song or url") + "]\n\n<b>" +
			utils.SC("supported platforms:") + "</b>\n- " + utils.SC("youtube") + "\n- " + utils.SC("spotify") +
			"\n- " + utils.SC("jiosaavn") + "\n- " + utils.SC("apple music"))
		_, _ = m.ReplyText(c, usageMsg, &td.SendMessageOpts{ReplyMarkup: utils.SupportKeyboard(), ParseMode: "HTML"})
		return td.EndGroups
	}

	updater, err := m.ReplyText(c, bq("🔍 "+utils.SC("searching and downloading...")), &td.SendMessageOpts{ParseMode: "HTML"})
	if err != nil {
		c.Logger.Warn("failed to send message", "error", err)
		return td.EndGroups
	}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(c, m, updater, rMsg, chatID, isVideo, force)
	}

	wrapper := downloader.NewDlWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			invalidMsg := bq(utils.SC("invalid url or unsupported platform.") + "\n\n<b>" +
				utils.SC("supported platforms:") + "</b>\n- " + utils.SC("youtube") + "\n- " + utils.SC("spotify") +
				"\n- " + utils.SC("jiosaavn") + "\n- " + utils.SC("apple music"))
			_, _ = updater.EditText(c, invalidMsg, &td.EditMessageTextOpts{ReplyMarkup: utils.SupportKeyboard(), ParseMode: "HTML"})
			return td.EndGroups
		}

		trackInfo, err := wrapper.GetInfo()
		if err != nil {
			_, _ = updater.EditText(c, bq(fmt.Sprintf("❌ <b>%s:</b> %s", utils.SC("error fetching track info"), html.EscapeString(err.Error()))), &td.EditMessageTextOpts{ParseMode: "HTML"})
			return td.EndGroups
		}

		if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
			_, _ = updater.EditText(c, bq(utils.SC("no tracks found.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
			return td.EndGroups
		}

		return handleUrl(c, m, updater, trackInfo, chatID, isVideo, force)
	}

	return handleTextSearch(c, m, updater, wrapper, chatID, isVideo, force)
}

// handleMedia handles playing media from a message.
func handleMedia(c *td.Client, m *td.Message, updater *td.Message, dlMsg *td.Message, chatId int64, isVideo bool, force bool) error {
	file, fileName := getFile(dlMsg)
	if file == nil {
		_, err := updater.EditText(c, bq(utils.SC("no valid media found in the message.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	if file.Size > config.MaxFileSize {
		_, err := updater.EditText(c, bq(fmt.Sprintf("<b>%s.</b> %s: %d MB.", utils.SC("file too large"), utils.SC("max size"), config.MaxFileSize/(1024*1024))), &td.EditMessageTextOpts{ParseMode: "HTML"})
		if err != nil {
			c.Logger.Warn("Edit message failed", "error", err)
		}
		return nil
	}

	fileId := dlMsg.RemoteFileID()
	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.EditText(c, bq(utils.SC("track already in queue or playing.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	dur := dlMsg.RemoteDuration()
	var linkURL string
	link, err := dlMsg.GetLink(c)
	if err != nil {
		c.Logger.Warn("Failed to get file link", "error", err)
	} else if link != nil {
		linkURL = link.Link
	}

	saveCache := utils.PlayerCache{
		URL: linkURL, Name: fileName, User: firstName(c, m), TrackID: fileId,
		Duration: dur, IsVideo: isVideo, Platform: utils.Telegram,
	}
	if audio, ok := dlMsg.Content.(*td.MessageAudio); ok && audio.Audio != nil {
		saveCache.Channel = audio.Audio.Performer
	}

	saveCache.Silent = messaging.SilentCommand(m)
	if queued, err := enqueueTrack(c, updater, chatId, &saveCache, force); queued {
		return err
	}

	filePath, err := downloader.DlCachedTrack(&saveCache, c, dlMsg)
	if err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, bq(fmt.Sprintf("<b>%s:</b> %s", utils.SC("download failed"), html.EscapeString(err.Error()))), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	if dur == 0 {
		dur = utils.GetMediaDuration(filePath)
		saveCache.Duration = dur
	}

	saveCache.FilePath = filePath

	if err = calls.Calls.PlayMedia(c, chatId, saveCache.FilePath, saveCache.IsVideo); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, bq(html.EscapeString(err.Error())), &td.EditMessageTextOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	return sendStartedStreaming(c, updater, &saveCache)
}

// handleTextSearch handles a text search for a song.
func handleTextSearch(c *td.Client, m *td.Message, updater *td.Message, wrapper *downloader.DlWrapper, chatId int64, isVideo bool, force bool) error {
	searchResult, err := wrapper.Search()
	if err != nil {
		_, err = updater.EditText(c, bq(fmt.Sprintf("❌ <b>%s:</b> %s", utils.SC("search failed"), html.EscapeString(err.Error()))), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.EditText(c, bq("😕 "+utils.SC("no results found. try a different query.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.Id); _track != nil {
		_, err := updater.EditText(c, bq(utils.SC("track already in queue or playing.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	return handleSingleTrack(c, m, updater, song, "", chatId, isVideo, force)
}

// handleUrl handles a URL search for a song.
func handleUrl(c *td.Client, m *td.Message, updater *td.Message, trackInfo *utils.PlatformTracks, chatId int64, isVideo bool, force bool) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.Id); _track != nil {
			_, err := updater.EditText(c, bq(utils.SC("track already in queue or playing.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
			return err
		}
		return handleSingleTrack(c, m, updater, track, "", chatId, isVideo, force)
	}

	return handleMultipleTracks(c, m, updater, trackInfo.Results, chatId, isVideo, force)
}

// handleSingleTrack handles a single track.
func handleSingleTrack(c *td.Client, m *td.Message, updater *td.Message, song utils.GetUrlTrack, filePath string, chatId int64, isVideo bool, force bool) error {
	if song.Duration > config.SongDurationLimit {
		_, err := updater.EditText(c, bq(fmt.Sprintf(utils.SC("sorry, song exceeds max duration of %d minutes."), config.SongDurationLimit/60)), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	saveCache := utils.PlayerCache{
		URL: song.Url, Name: song.Title, User: firstName(c, m), FilePath: filePath,
		Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration, Channel: song.Channel, Views: song.Views,
		IsVideo: isVideo, Platform: song.Platform,
	}

	saveCache.Silent = messaging.SilentCommand(m)
	if queued, err := enqueueTrack(c, updater, chatId, &saveCache, force); queued {
		return err
	}

	if saveCache.FilePath == "" {
		dlResult, err := downloader.DlCachedTrack(&saveCache, c, nil)
		if err != nil {
			cache.ChatCache.RemoveCurrentSong(chatId)
			_, err = updater.EditText(c, bq(fmt.Sprintf("<b>%s:</b> %s", utils.SC("download failed"), html.EscapeString(err.Error()))), &td.EditMessageTextOpts{ParseMode: "HTML"})
			return err
		}

		saveCache.FilePath = dlResult
	}

	if err := calls.Calls.PlayMedia(c, chatId, saveCache.FilePath, saveCache.IsVideo); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, bq(html.EscapeString(err.Error())), &td.EditMessageTextOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	if err := sendStartedStreaming(c, updater, &saveCache); err != nil {
		c.Logger.Warn("Edit message failed", "error", err)
		return err
	}

	return nil
}
// handleMultipleTracks handles multiple tracks.
func handleMultipleTracks(c *td.Client, m *td.Message, updater *td.Message, tracks []utils.GetUrlTrack, chatId int64, isVideo bool, force bool) error {
	if len(tracks) == 0 {
		_, err := updater.EditText(c, bq(utils.SC("no tracks found.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	queueHeader := "<blockquote><u><b>" + utils.SC("added to queue:") + "</b></u>\n<blockquote expandable>\n"
	var tracksToAdd []*utils.PlayerCache
	var skippedTracks []string

	for _, track := range tracks {
		if track.Duration > config.SongDurationLimit {
			skippedTracks = append(skippedTracks, track.Title)
			continue
		}

		saveCache := &utils.PlayerCache{
			Name: track.Title, TrackID: track.Id, Duration: track.Duration,
			Thumbnail: track.Thumbnail, User: firstName(c, m), Platform: track.Platform,
			IsVideo: isVideo, URL: track.Url, Channel: track.Channel, Views: track.Views,
		}
		saveCache.Silent = messaging.SilentCommand(m)
		tracksToAdd = append(tracksToAdd, saveCache)
	}

	if len(tracksToAdd) == 0 {
		if len(skippedTracks) > 0 {
			_, err := updater.EditText(c, bq(fmt.Sprintf(utils.SC("all tracks were skipped (max duration %d min)."), config.SongDurationLimit/60)), &td.EditMessageTextOpts{ParseMode: "HTML"})
			return err
		}
		_, err := updater.EditText(c, bq(utils.SC("no valid tracks found.")), &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}

	var qLenAfter int
	var startLen int

	if force {
		qLenAfter = 0
		for _, t := range slices.Backward(tracksToAdd) {
			qLenAfter = cache.ChatCache.AddSongToFront(chatId, t)
		}
		startLen = qLenAfter - len(tracksToAdd)
		if startLen > 0 {
			_ = calls.Calls.PlayNext(c, chatId)
			_ = c.DeleteMessages(chatId, []int64{updater.Id}, &td.DeleteMessagesOpts{Revoke: true})
			return nil
		}
	} else {
		qLenAfter = cache.ChatCache.AddSongs(chatId, tracksToAdd)
		startLen = qLenAfter - len(tracksToAdd)
	}

	if startLen == 0 {
		return calls.Calls.PlayTrackWithMessage(c, updater, chatId, tracksToAdd[0])
	}

	var sb strings.Builder
	sb.WriteString(queueHeader)

	var totalDuration int32
	for i, track := range tracksToAdd {
		currentQLen := startLen + i + 1
		escTrackName := html.EscapeString(track.Name)
		fmt.Fprintf(&sb, "<b>%d.</b> %s\n└ <b>%s:</b> %s\n",
			currentQLen, escTrackName, utils.SC("duration"), utils.SecToMin(track.Duration))
		totalDuration += track.Duration
	}

	sb.WriteString("</blockquote>")
	escRequester := html.EscapeString(firstName(c, m))
	queueSummary := fmt.Sprintf(
		"\n<b>%s:</b> %d\n<b>%s:</b> %s ᴍɪɴ\n<b>%s:</b> %s\n</blockquote>",
		utils.SC("queue total"), qLenAfter,
		utils.SC("duration"), utils.SecToMin(totalDuration),
		utils.SC("requested by"), escRequester,
	)

	sb.WriteString(queueSummary)
	if len(skippedTracks) > 0 {
		fmt.Fprintf(&sb, "\n\n<blockquote><b>%s %d %s</b> (%s).</blockquote>",
			utils.SC("skipped"), len(skippedTracks), utils.SC("tracks"), utils.SC("exceeded duration limit"))
	}

	fullMessage := sb.String()

	if len(fullMessage) > 4096 {
		fullMessage = "<blockquote>" + strings.TrimPrefix(queueSummary, "\n")
	}

	_, err := messaging.FinishProgress(c, updater, fullMessage, &td.EditMessageTextOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           utils.QueueMarkup(tracksToAdd[0].TrackID),
		DisableWebPagePreview: true,
	}, messaging.SilentCommand(m))

	return err
}

func enqueueTrack(c *td.Client, updater *td.Message, chatId int64, saveCache *utils.PlayerCache, force bool) (bool, error) {
	var qLen int
	if force {
		qLen = cache.ChatCache.AddSongToFront(chatId, saveCache)
	} else {
		qLen = cache.ChatCache.AddSong(chatId, saveCache)
	}

	if qLen > 1 {
		if force {
			_ = calls.Calls.PlayNext(c, chatId)
			_ = c.DeleteMessages(chatId, []int64{updater.Id}, &td.DeleteMessagesOpts{Revoke: true})
			return true, nil
		}
		escURL := html.EscapeString(saveCache.URL)
		escName := html.EscapeString(saveCache.Name)
		escUser := html.EscapeString(saveCache.User)
		queueInfo := fmt.Sprintf(
			"<blockquote><u><b>%s %d</b></u>\n\n<b>%s:</b> <a href='%s'>%s</a>\n\n<b>%s:</b> %s ᴍɪɴ\n<b>%s:</b> %s</blockquote>",
			utils.SC("added to queue:"), qLen,
			utils.SC("title"), escURL, escName,
			utils.SC("duration"), utils.SecToMin(saveCache.Duration),
			utils.SC("requested by"), escUser,
		)

		_, err := messaging.FinishProgress(c, updater, queueInfo, &td.EditMessageTextOpts{
			ReplyMarkup:           utils.QueueMarkup(saveCache.TrackID),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		}, saveCache.Silent)
		return true, err
	}

	return false, nil
}

func sendStartedStreaming(c *td.Client, updater *td.Message, saveCache *utils.PlayerCache) error {
	_, err := messaging.SendPanel(c, updater, saveCache, saveCache.Silent)
	return err
}
