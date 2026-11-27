/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package bot

import (
	"fmt"
	"html"
	"strings"

	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

type helpCat struct {
	Title   string
	Content string
	Markup  *td.ReplyMarkupInlineKeyboard
}

// helpTable renders rows of {command, description}.
func helpTable(intro string, rows ...[2]string) string {
	var sb strings.Builder
	if intro != "" {
		sb.WriteString("<p>" + html.EscapeString(intro) + "</p>")
	}
	sb.WriteString("<table bordered striped><tr><th>Command</th><th>Description</th></tr>")
	for _, r := range rows {
		sb.WriteString("<tr><td><code>" + html.EscapeString(r[0]) + "</code></td><td>" + html.EscapeString(r[1]) + "</td></tr>")
	}
	sb.WriteString("</table>")
	return sb.String()
}

func getHelpCategories() map[string]helpCat {
	back := utils.BackHelpMenuKeyboard()
	mk := func(title, content string) helpCat { return helpCat{Title: utils.SC(title), Content: content, Markup: back} }

	return map[string]helpCat{
		"help_admin": mk("admins", helpTable("Playback controls for chat admins and authorized users.",
			[2]string{"/skip", "Skip the current track."},
			[2]string{"/pause  /resume", "Pause or resume playback."},
			[2]string{"/mute  /unmute", "Mute or unmute the assistant."},
			[2]string{"/seek [sec]", "Jump forward in the track."},
			[2]string{"/remove [n]", "Remove a track from the queue."},
			[2]string{"/end  /stop", "Stop playback and clear the queue."},
			[2]string{"/queue", "Show the current queue."},
			[2]string{"/reload", "Refresh the admin cache."},
			[2]string{"/settings", "Play mode, admin mode, auto-delete, silent mode, language."},
			[2]string{"/join  /link", "Make the assistant join this chat."},
		)),
		"help_auth": mk("auth", helpTable("Authorized users can use admin commands without being admins.",
			[2]string{"/auth [reply|id]", "Authorize a user in this chat."},
			[2]string{"/rmauth [reply|id]", "Remove authorization."},
			[2]string{"/authlist", "List authorized users."},
		)),
		"help_blacklist": mk("blacklist", helpTable("Bot owner / devs only.",
			[2]string{"/blacklist [id]", "Block a chat (negative id) or a user (positive id)."},
			[2]string{"/unblacklist [id]", "Remove a chat or user from the blacklist."},
		)),
		"help_language": mk("language", helpTable("",
			[2]string{"/lang", "Open settings to change the chat language."},
			[2]string{"/settings", "All chat settings including language."},
		)),
		"help_ping": mk("ping", helpTable("",
			[2]string{"/ping", "Show latency, uptime and system info."},
			[2]string{"/start", "Start the bot."},
			[2]string{"/help", "Open this menu."},
		)),
		"help_play": mk("play", helpTable("Play from YouTube, Spotify, SoundCloud, Apple Music and links.",
			[2]string{"/play [song|link]", "Play audio in the voice chat."},
			[2]string{"/vplay [song|link]", "Play video."},
			[2]string{"/fplay  /fvplay", "Force play, skipping the queue."},
			[2]string{"/mix [song]", "Queue a mix of related songs."},
			[2]string{"/splay  /sskip ...", "s-prefix runs a command silently."},
		)),
		"help_stats": mk("stats", helpTable("Bot owner / devs only.",
			[2]string{"/stats", "Bot statistics."},
			[2]string{"/bdata", "Today's and last 24h groups added / removed, totals."},
			[2]string{"/vc", "Active voice chat status (audio, video, stuck)."},
			[2]string{"/av", "List active voice chats."},
		)),
		"help_sudoers": mk("sudoers", helpTable("Bot owner / devs only.",
			[2]string{"/broadcast", "Reply to a message to broadcast it (-chat -user -copy)."},
			[2]string{"/stop_gcast", "Stop a running broadcast."},
			[2]string{"/selfpromo run", "Send the promo post to all groups now (also sent daily at 7 AM IST)."},
			[2]string{"/logger", "Toggle the play logger."},
			[2]string{"/clearass  /leaveall", "Assistant maintenance."},
		)),
		"help_autoplay": mk("autoplay", helpTable("Keep music going with recommended tracks. When the queue ends and autoplay is off, the VC is closed; playing again starts it back up.",
			[2]string{"/autoplay", "Toggle autoplay (also a button on the player)."},
		)),
		"help_vclogger": mk("vc logger", helpTable("Announces who joins or leaves the voice chat (auto-deletes after 7s).",
			[2]string{"/vclogger on|off", "Enable or disable the VC logger (admins)."},
		)),
		"help_loop": mk("loop", helpTable("",
			[2]string{"/loop [0-10]", "Repeat the current track N times (0 to disable)."},
		)),
		"help_playlist": mk("playlist", helpTable("Personal playlists.",
			[2]string{"/createplaylist", "Create a playlist."},
			[2]string{"/deleteplaylist", "Delete a playlist."},
			[2]string{"/addtoplaylist", "Add a track."},
			[2]string{"/removefromplaylist", "Remove a track."},
			[2]string{"/playlistinfo", "Playlist details."},
			[2]string{"/myplaylists", "List your playlists."},
		)),
		"help_restrictor": mk("restrictor", helpTable("Keeps your chat clean. Admins, auth users and approved users are never touched. Make me admin with delete rights.",
			[2]string{"/linkremove on|off", "Delete messages that contain links (alias /antilink)."},
			[2]string{"/bioremove on|off", "Delete messages from users who have a link in their bio (alias /antibio)."},
		)),
		"help_approve": mk("approve", helpTable("Approved users are exempt from the link and bio remover.",
			[2]string{"/approve [reply|id]", "Approve a user."},
			[2]string{"/unapprove [reply|id]", "Remove approval."},
			[2]string{"/approved", "List approved users."},
		)),
	}
}

func helpCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "User", Id: cb.SenderUserId}
	}

	edit := func(text string, markup *td.ReplyMarkupInlineKeyboard) {
		rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: text}}
		_, _ = c.EditRichMessage(cb.ChatId, cb.MessageId, rich, &td.EditRichMessageOpts{ReplyMarkup: markup})
	}

	if strings.Contains(data, "help_all") {
		_ = cb.Answer(c, "Opening help menu...", nil)
		edit(fmt.Sprintf("<h3>✨ %s, %s!</h3><p>%s</p>",
			utils.SC("welcome"), html.EscapeString(user.FirstName),
			utils.SC("select a category below to browse the commands.")), utils.HelpMenuKeyboard())
		return nil
	}

	if strings.Contains(data, "help_back") {
		_ = cb.Answer(c, "Returning to main menu...", nil)
		edit(startText(user.FirstName), utils.AddMeMarkup(c.Me().Usernames.EditableUsername))
		return nil
	}

	if category, ok := getHelpCategories()[data]; ok {
		_ = cb.Answer(c, category.Title, nil)
		edit(fmt.Sprintf("<h3>%s</h3>%s", category.Title, category.Content), category.Markup)
		return nil
	}

	_ = cb.Answer(c, "Unknown help category.", &td.CallbackAnswerOpts{ShowAlert: true})
	return nil
}
