package utils

import (
	"fmt"
	"html"
	"math/rand/v2"
	"strings"

	"istkharmusic/internal/config"

	td "github.com/AshokShau/gotdbot"
)

func urlBtn(text, link string) td.InlineKeyboardButton {
	return td.InlineKeyboardButton{
		Text: text,
		Type: &td.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
	}
}

func cbBtn(text, data string) td.InlineKeyboardButton {
	return td.InlineKeyboardButton{
		Text: text,
		Type: &td.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
	}
}

// URLButton builds a URL inline button.
func URLButton(text, link string) td.InlineKeyboardButton {
	return urlBtn(text, link)
}

// CBButton builds a callback inline button.
func CBButton(text, data string) td.InlineKeyboardButton {
	return cbBtn(text, data)
}

func randomStyles() [3]string {
	st := [3]string{"primary", "success", "danger"}
	rand.Shuffle(len(st), func(i, j int) { st[i], st[j] = st[j], st[i] })
	return st
}

// ProgressBar renders "1:00 ─────●─── 4:38" (elapsed, bar, total).
func ProgressBar(played, total int32) string {
	const slots = 9
	pos := 0
	if total > 0 && played > 0 {
		pos = int(int64(played) * slots / int64(total))
	}
	if pos < 0 {
		pos = 0
	}
	if pos > slots-1 {
		pos = slots - 1
	}
	var b strings.Builder
	for i := 0; i < slots; i++ {
		if i == pos {
			b.WriteString("●")
		} else {
			b.WriteString("─")
		}
	}
	return fmt.Sprintf("%s %s %s", SecToMin(played), b.String(), SecToMin(total))
}

// PanelButtons builds the now-playing inline keyboard matching internal/messaging/panel.go.
func PanelButtons(chatID int64, mode string, played, total int32, queue int, autoplay bool) *td.ReplyMarkupInlineKeyboard {
	toggleText := "II ᴘᴀᴜꜱᴇ"
	toggleData := fmt.Sprintf("admin_pause:%d", chatID)
	if mode == "pause" {
		toggleText = "▷ ʀᴇꜱᴜᴍᴇ"
		toggleData = fmt.Sprintf("admin_resume:%d", chatID)
	}

	if queue < 0 {
		queue = 0
	}

	apText := "ᴀᴜᴛᴏᴘʟᴀʏ: ᴏꜰꜰ"
	if autoplay {
		apText = "ᴀᴜᴛᴏᴘʟᴀʏ: ᴏɴ"
	}

	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				cbBtn(ProgressBar(played, total), "get_timer"),
			},
			{
				cbBtn("↺ ʀᴇᴘʟᴀʏ", fmt.Sprintf("admin_replay:%d", chatID)),
				cbBtn(toggleText, toggleData),
				cbBtn("» ꜱᴋɪᴘ", fmt.Sprintf("admin_skip:%d", chatID)),
			},
			{
				cbBtn(fmt.Sprintf("☰ ǫᴜᴇᴜᴇ · %d", queue), fmt.Sprintf("play_queue:%d", chatID)),
				cbBtn("▢ ᴇɴᴅ ǫᴜᴇᴜᴇ", fmt.Sprintf("admin_stop:%d", chatID)),
			},
			{
				cbBtn(apText, fmt.Sprintf("admin_autoplay:%d", chatID)),
			},
		},
	}
}

// ControlButtons builds compact playback control buttons for panels.
func ControlButtons(chatID int64, paused bool) *td.ReplyMarkupInlineKeyboard {
	toggleText := "II ᴘᴀᴜꜱᴇ"
	toggleData := fmt.Sprintf("admin_pause:%d", chatID)
	if paused {
		toggleText = "▷ ʀᴇꜱᴜᴍᴇ"
		toggleData = fmt.Sprintf("admin_resume:%d", chatID)
	}

	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				cbBtn("↺ ʀᴇᴘʟᴀʏ", fmt.Sprintf("admin_replay:%d", chatID)),
				cbBtn(toggleText, toggleData),
				cbBtn("» ꜱᴋɪᴘ", fmt.Sprintf("admin_skip:%d", chatID)),
				cbBtn("▢ ᴇɴᴅ ǫᴜᴇᴜᴇ", fmt.Sprintf("admin_stop:%d", chatID)),
			},
		},
	}
}

func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

const (
	headerPlaying = "ᴍᴜꜱɪᴄ ɪꜱ ᴘʟᴀʏɪɴɢ"
	footerLine1   = "ʏᴏᴜʀ ᴛʀᴀᴄᴋ ɪꜱ ɴᴏᴡ ᴘʟᴀʏɪɴɢ"
	footerLine2   = "ꜱɪᴛ ʙᴀᴄᴋ, ʀᴇʟᴀx &amp; ᴇɴᴊᴏʏ ᴛʜᴇ ᴍᴜꜱɪᴄ"
)

func panelHTML(img, header, title, link, duration, requestedBy, extra string) string {
	var b strings.Builder
	if img != "" {
		fmt.Fprintf(&b, `<img src="%s"/>`, html.EscapeString(img))
	}
	fmt.Fprintf(&b, "<p>🎧 %s</p>", html.EscapeString(header))
	b.WriteString("<p>")
	fmt.Fprintf(&b, "🎵 %s : <a href=\"%s\">%s</a><br>", SC("now playing"), html.EscapeString(link), html.EscapeString(title))
	fmt.Fprintf(&b, "⏱️ %s : %s ᴍɪɴ<br>", SC("track duration"), html.EscapeString(duration))
	fmt.Fprintf(&b, "👤 %s : %s", SC("requested by"), html.EscapeString(requestedBy))
	if extra != "" {
		fmt.Fprintf(&b, "<br>%s", html.EscapeString(extra))
	}
	b.WriteString("</p>")
	fmt.Fprintf(&b, "<p>🎶 %s<br>🔊 %s</p>", footerLine1, footerLine2)
	return b.String()
}

// RenderPlayPanelText is the fallback card when no track information is available.
func RenderPlayPanelText(title, link, duration, requestedBy string) string {
	return panelHTML("", headerPlaying, title, link, duration, requestedBy, "")
}

// NowPlayingHTML builds the now-playing panel HTML.
func NowPlayingHTML(song *PlayerCache, status, extra string) string {
	if song == nil {
		return RenderPlayPanelText("Unknown Track", "https://t.me", "0:00", "Unknown")
	}
	header := status
	if status == "" || status == SC("now playing") || status == SC("streaming started | enjoy music") {
		header = headerPlaying
	}
	img := config.ImgOr(config.PlayImg, song.Thumbnail)
	return panelHTML(img, header, shorten(song.Name, 60), song.URL, SecToMin(song.Duration), song.User, extra)
}
