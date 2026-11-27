package utils

import (
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func RenderPlayPanelText(title, url, duration, requestedBy string) string {
	safeTitle := escapeHTML(title)
	safeURL := escapeHTML(url)
	safeBy := escapeHTML(requestedBy)

	return fmt.Sprintf(`<blockquote>🎧 <b>ᴍᴜꜱɪᴄ ɪꜱ ᴘʟᴀʏɪɴɢ</b>

🎵 <b>ɴᴏᴡ ᴘʟᴀʏɪɴɢ :</b> <a href="%s">%s</a>
⏱️ <b>ᴛʀᴀᴄᴋ ᴅᴜʀᴀᴛɪᴏɴ :</b> %s ᴍɪɴ
👤 <b>ʀᴇǫᴜᴇꜱᴛᴇᴅ ʙʏ :</b> %s

🎶 ʏᴏᴜʀ ᴛʀᴀᴄᴋ ɪꜱ ɴᴏᴡ ᴘʟᴀʏɪɴɢ
🔊 ꜱɪᴛ ʙᴀᴄᴋ, ʀᴇʟᴀx & ᴇɴᴊᴏʏ ᴛʜᴇ ᴍᴜꜱɪᴄ</blockquote>`, safeURL, safeTitle, duration, safeBy)
}

func NowPlayingHTML(track *PlayerCache, status, extra string) string {
	if track == nil {
		return RenderPlayPanelText("Unknown Track", "https://t.me", "0:00", "Unknown")
	}
	base := fmt.Sprintf(`<blockquote>🎧 <b>%s</b>

🎵 <b>%s :</b> <a href="%s">%s</a>
⏱️ <b>%s :</b> %s ᴍɪɴ
👤 <b>%s :</b> %s`,
		html.EscapeString(status),
		html.EscapeString(SC("now playing")),
		html.EscapeString(track.URL),
		html.EscapeString(track.Name),
		html.EscapeString(SC("track duration")),
		html.EscapeString(SecToMin(track.Duration)),
		html.EscapeString(SC("requested by")),
		html.EscapeString(track.User),
	)
	if extra != "" {
		base += "\n" + html.EscapeString(extra)
	}
	return base + "</blockquote>"
}

func PanelButtons(mode string, played int32, duration int32, autoplay bool) *td.ReplyMarkupInlineKeyboard {
	autoLabel := SC("autoplay: off")
	if autoplay {
		autoLabel = SC("autoplay: on")
	}

	midBtnText := SC("pause")
	midBtnData := "play_pause"
	if mode == "pause" {
		midBtnText = SC("resume")
		midBtnData = "play_resume"
	}

	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{Text: SC("skip"), Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("play_skip")}},
				{Text: midBtnText, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(midBtnData)}},
				{Text: SC("stop"), Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("play_stop")}},
			},
			{
				{Text: autoLabel, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("play_autoplay")}},
			},
		},
	}
}
