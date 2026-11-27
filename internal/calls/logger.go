package calls

import (
	"fmt"
	"html"
	"time"

	"istkharmusic/internal/config"
	"istkharmusic/internal/db"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

func istNow() string {
	loc := time.FixedZone("IST", 5*3600+30*60)
	return time.Now().In(loc).Format("02 Jan 2006 • 15:04")
}

func chatTitle(client *td.Client, chatID int64) string {
	if ch, err := client.GetChat(chatID); err == nil && ch != nil && ch.Title != "" {
		return html.EscapeString(ch.Title)
	}
	return "Unknown Group"
}

func sendLogger(client *td.Client, chatID int64, song *utils.PlayerCache) {
	if chatID == 0 || song == nil || config.LoggerId == 0 || chatID == config.LoggerId {
		return
	}
	if !db.Instance.GetLoggerStatus() {
		return
	}

	title := html.EscapeString(song.Name)
	if song.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(song.URL), title)
	}

	text := fmt.Sprintf(
		"🎧 <b>%s • #PLAYBACK_STARTED</b>\n\n"+
			"<blockquote>🎵 <b>%s:</b> %s\n"+
			"⏱ <b>%s:</b> %s\n"+
			"📡 <b>%s:</b> %s\n"+
			"👥 <b>%s:</b> %s (<code>%d</code>)\n"+
			"👤 <b>%s:</b> %s\n"+
			"🗓 <b>%s:</b> %s</blockquote>",
		utils.SC("playback started"),
		utils.SC("track"), title,
		utils.SC("duration"), utils.SecToMin(song.Duration),
		utils.SC("source"), html.EscapeString(song.Platform),
		utils.SC("group"), chatTitle(client, chatID), chatID,
		utils.SC("requested by"), html.EscapeString(song.User),
		utils.SC("time"), istNow(),
	)

	_, err := client.SendMessage(config.LoggerId, text, &td.SendMessageOpts{
		DisableWebPagePreview: true,
		ParseMode:             "HTML",
	})
	if err != nil {
		logger.Warn("Failed to send playback start log", "error", err)
	}
}

// SendQueueLog logs #PLAYBACK_QUEUED when a song is added to queue.
func SendQueueLog(client *td.Client, chatID int64, song *utils.PlayerCache) {
	if chatID == 0 || song == nil || config.LoggerId == 0 || chatID == config.LoggerId {
		return
	}
	if !db.Instance.GetLoggerStatus() {
		return
	}

	title := html.EscapeString(song.Name)
	if song.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(song.URL), title)
	}

	text := fmt.Sprintf(
		"📥 <b>%s • #PLAYBACK_QUEUED</b>\n\n"+
			"<blockquote>🎵 <b>%s:</b> %s\n"+
			"👥 <b>%s:</b> %s (<code>%d</code>)\n"+
			"👤 <b>%s:</b> %s</blockquote>",
		utils.SC("playback queued"),
		utils.SC("track"), title,
		utils.SC("group"), chatTitle(client, chatID), chatID,
		utils.SC("requested by"), html.EscapeString(song.User),
	)

	_, err := client.SendMessage(config.LoggerId, text, &td.SendMessageOpts{
		DisableWebPagePreview: true,
		ParseMode:             "HTML",
	})
	if err != nil {
		logger.Warn("Failed to send queue log", "error", err)
	}
}

// OnVcParticipant is called when a user joins/leaves a voice chat the assistant is in.
var OnVcParticipant func(chatID, userID int64, joined bool)
