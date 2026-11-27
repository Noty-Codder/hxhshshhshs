/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package bot

import (
	"istkharmusic/internal/calls"
	"istkharmusic/internal/messaging"
	"time"

	"github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/filters/callbackquery"
)

var startTime = time.Now()

var botClient *gotdbot.Client

func LoadModules(c *gotdbot.Client, cancelCleanup func(int64)) {
	botUser := c.Me().Usernames.EditableUsername
	// on registers /name and /name@BotUsername (needed in groups with many bots).
	on := func(name string, handler func(*gotdbot.Client, *gotdbot.Message) error) {
		c.OnCommand(name, handler)
		if botUser != "" {
			c.OnCommand(name+"@"+botUser, handler)
		}
	}
	register := func(name string, handler func(*gotdbot.Client, *gotdbot.Message) error) {
		on(name, handler)
		if messaging.SupportsSilence(name) {
			on("s"+name, handler)
		}
	}
	botClient = c
	calls.OnVcParticipant = func(chatID, userID int64, joined bool) { vcParticipantEvent(c, chatID, userID, joined) }

	on("bdata", bdataHandler)
	on("vc", vcStatusHandler)
	on("selfpromo", selfPromoHandler)
	on("blacklist", blacklistHandler(true))
	on("unblacklist", blacklistHandler(false))
	on("linkremove", linkRemoveHandler)
	on("antilink", linkRemoveHandler)
	on("bioremove", bioRemoveHandler)
	on("antibio", bioRemoveHandler)
	on("vclogger", vcLoggerToggleHandler)
	on("approve", approveHandler)
	on("unapprove", unapproveHandler)
	on("approved", approvedListHandler)
	on("lang", settingsHandler)
	c.AddUpdateNewMessageHandlerGroup(restrictWatcher, nil, -3)
	c.AddUpdateNewMessageHandlerGroup(plainTriggerWatcher, nil, -4)
	on("reload", reloadAdminCacheHandler)
	on("join", joinHandler)
	on("link", joinHandler)
	on("authList", authListHandler)
	on("auths", authListHandler)
	on("auth", addAuthHandler)
	on("addAuth", addAuthHandler)
	on("removeAuth", removeAuthHandler)
	on("rmAuth", removeAuthHandler)
	on("broadcast", broadcastHandler)
	on("gCast", broadcastHandler)
	on("stop_gcast", cancelBroadcastHandler)
	on("stop_broadcast", cancelBroadcastHandler)
	on("av", activeVcHandler)
	on("active_vc", activeVcHandler)
	on("clearass", clearAssistantsHandler)
	on("clearAssistants", clearAssistantsHandler)
	on("leaveAll", leaveAllHandler)
	on("logger", loggerHandler)
	on("privacy", privacyHandler)
	on("autoplay", autoplayHandler)
	register("loop", loopHandler)
	register("pause", pauseHandler)
	register("resume", resumeHandler)
	on("cplist", createPlaylistHandler)
	on("createplaylist", createPlaylistHandler)
	on("deleteplaylist", deletePlaylistHandler)
	register("queue", queueHandler)
	register("seek", seekHandler)
	on("sh", shellCommand)
	register("skip", skipHandler)
	register("stop", stopHandler)
	register("end", stopHandler)
	on("start", startHandler)
	on("help", startHandler)
	on("ping", pingHandler)
	register("mix", mixHandler)
	register("play", playHandler)
	register("p", playHandler)
	register("fplay", fPlayHandler)
	register("fp", fPlayHandler)
	register("vplay", vPlayHandler)
	register("v", vPlayHandler)
	register("fvplay", fVPlayHandler)
	register("fvp", fVPlayHandler)
	register("remove", removeHandler)
	register("mute", muteHandler)
	register("unmute", unmuteHandler)
	on("settings", settingsHandler)
	on("addtoplaylist", addToPlaylistHandler)
	on("addtoplist", addToPlaylistHandler)
	on("removefromplaylist", removeFromPlaylistHandler)
	on("rmplist", removeFromPlaylistHandler)
	on("plistinfo", playlistInfoHandler)
	on("playlistinfo", playlistInfoHandler)
	on("myplaylists", myPlaylistsHandler)
	on("myplist", myPlaylistsHandler)
	on("stats", statsHandler)

	c.OnUpdateNewCallbackQuery(helpCallbackHandler, callbackquery.Prefix("help_"))
	c.OnUpdateNewCallbackQuery(playCallbackHandler, callbackquery.Prefix("play_"))
	c.OnUpdateNewCallbackQuery(vcPlayHandler, callbackquery.Prefix("vcplay_"))
	c.OnUpdateNewCallbackQuery(func(client *gotdbot.Client, cb *gotdbot.UpdateNewCallbackQuery) error {
		return settingsCallbackHandler(client, cb, cancelCleanup)
	}, callbackquery.Prefix("settings_"))
	c.OnUpdateNewCallbackQuery(autoplayCallbackHandler, callbackquery.Equal("autoplay_toggle"))

	c.OnUpdateChatMember(handleParticipant, nil)
	c.OnUpdateNewChat(handleNewChat, nil)
	c.OnUpdateNewMessage(handleVoiceChatMessage, nil)

	c.Logger.Debug("Handlers loaded successfully")
}
