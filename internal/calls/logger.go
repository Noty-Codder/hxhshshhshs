/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package calls

import (
	"istkharmusic/internal/config"
	"istkharmusic/internal/utils"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func sendLogger(client *td.Client, chatID int64, song *utils.PlayerCache) {
	if chatID == 0 || song == nil || chatID == config.LoggerId {
		return
	}

	text := fmt.Sprintf(
		"<b>A song is playing</b> in <code>%d</code>\n\n‣ <b>Title:</b> <a href='%s'>%s</a>\n‣ <b>Duration:</b> %s\n‣ <b>Requested by:</b> %s\n‣ <b>Platform:</b> %s\n‣ <b>Is Video:</b> %t",
		chatID,
		song.URL,
		song.Name,
		utils.SecToMin(song.Duration),
		song.User,
		song.Platform,
		song.IsVideo,
	)

	_, err := client.SendMessage(config.LoggerId, text, &td.SendMessageOpts{DisableWebPagePreview: true, ParseMode: "HTML"})
	if err != nil {
		logger.Warn("Failed to send the message", "error", err)
	}
}

// OnVcParticipant is called when a user joins/leaves a voice chat the assistant is in.
var OnVcParticipant func(chatID, userID int64, joined bool)
