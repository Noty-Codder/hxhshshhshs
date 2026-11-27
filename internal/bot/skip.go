/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package bot

import (
	"istkharmusic/internal/cache"
	"istkharmusic/internal/calls"
	"istkharmusic/internal/messaging"

	td "github.com/AshokShau/gotdbot"
)

func skipHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return nil
	}

	_ = calls.Calls.PlayNext(c, chatID, messaging.SilentCommand(m))
	return nil
}
