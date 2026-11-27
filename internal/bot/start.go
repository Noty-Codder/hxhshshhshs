/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package bot

import (
	"istkharmusic/internal/config"
	"istkharmusic/internal/messaging"
	"istkharmusic/internal/utils"
	"fmt"
	"html"
	"runtime"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func pingHandler(c *td.Client, m *td.Message) error {
	deleteCmd(c, m)

	start := time.Now()

	msg, err := m.ReplyText(c, utils.SC("pinging… please wait…"), nil)
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := getFormattedDuration(time.Since(startTime))

	response := fmt.Sprintf(
		`<img src="%s"/><blockquote><b>🏓 %s</b><br>⚡ <b>%s:</b> <code>%d ms</code><br>⏱ <b>%s:</b> <code>%s</code><br>🧵 <b>%s:</b> <code>%d</code></blockquote>`,
		config.ImgOr(config.PingImg, ""),
		utils.SC("pong! system status"),
		utils.SC("latency"), latency,
		utils.SC("uptime"), uptime,
		utils.SC("go routines"), runtime.NumGoroutine(),
	)

	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: response}}
	if _, err = m.ReplyRichMessage(c, rich, nil); err != nil {
		plain := fmt.Sprintf("<b>📊 %s</b>\n\n<b>%s:</b> <code>%d ms</code>\n<b>%s:</b> <code>%s</code>",
			utils.SC("system status"), utils.SC("latency"), latency, utils.SC("uptime"), uptime)
		_, err = msg.EditText(c, plain, &td.EditMessageTextOpts{ParseMode: "HTML"})
		return err
	}
	_ = msg.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
	return nil
}

// startText builds the premium start message (rich HTML).
func startText(name string) string {
	return fmt.Sprintf(`<img src="%s"/>
<h3>✨ %s, %s!</h3>
<p><b>%s</b> — %s</p>
<blockquote>⚡ %s<br>🎧 %s<br>📺 %s<br>🔁 %s<br>🛡 %s<br>🌐 %s</blockquote>
<p>%s</p>`,
		config.Img(config.StartImg),
		utils.SC("welcome"), html.EscapeString(name),
		utils.SC(config.BotName), utils.SC("premium music experience"),
		utils.SC("fast and stable streaming"),
		utils.SC("high quality audio"),
		utils.SC("video playback (/vplay)"),
		utils.SC("autoplay • queue • playlists"),
		utils.SC("link / bio restrictor"),
		"YouTube • Spotify • SoundCloud • Apple Music",
		utils.SC("add me to your group, then send /play song name."),
	)
}

func startHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	go storeChatToDB(chatID)

	deleteCmd(c, m)

	if m.IsPrivate() {
		rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: startText(firstName(c, m))}}
		_, err := m.ReplyRichMessage(c, rich, &td.SendMessageOpts{
			ReplyMarkup: utils.AddMeMarkup(c.Me().Usernames.EditableUsername),
		})
		return err
	}

	uptime := getFormattedDuration(time.Since(startTime))
	htmlText := fmt.Sprintf(`<img src="%s"/><blockquote><b>🎵 %s %s</b><br>⏱ <b>%s:</b> <code>%s</code></blockquote>`,
		config.Img(config.StartImg), utils.SC(config.BotName), utils.SC("is ready!"), utils.SC("uptime"), uptime)

	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: htmlText}}
	_, err := m.ReplyRichMessage(c, rich, &td.SendMessageOpts{ReplyMarkup: utils.SupportKeyboard()})
	return err
}

// sendWelcome posts the "thanks for adding" panel when the bot joins a group.
func sendWelcome(c *td.Client, chatID, actorID int64) {
	adder := "Unknown"
	if actorID > 0 {
		if u, err := c.GetUser(actorID); err == nil && u != nil {
			adder = u.FirstName
		}
	}
	title := "this group"
	if ch, err := c.GetChat(chatID); err == nil && ch != nil {
		title = ch.Title
	}
	text := fmt.Sprintf(`<img src="%s"/><blockquote>🎵 <b>%s</b> @%s<br><br>👤 <b>%s:</b> %s<br>📌 <b>%s:</b> %s<br>🆔 <b>%s:</b> <code>%d</code><br><br>%s<br>%s<br><br>/play %s — %s</blockquote>`,
		config.ImgOr(config.WelcomeImg, ""),
		utils.SC("thanks for adding"), c.Me().Usernames.EditableUsername,
		utils.SC("added by"), html.EscapeString(adder),
		utils.SC("group"), html.EscapeString(title),
		utils.SC("chat id"), chatID,
		utils.SC("i can play music in this group voice chat."),
		utils.SC("to change language, send /lang."),
		utils.SC("song name"), utils.SC("play a song"))
	if _, err := messaging.SendRich(c, chatID, text, utils.SupportKeyboard()); err != nil {
		c.Logger.Warn("welcome message failed", "chat_id", chatID, "error", err)
	}
}
