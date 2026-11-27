package utils

import (
	"fmt"
	"html"
	"math/rand/v2"
	"strings"

	"istkharmusic/internal/config"

	"github.com/AshokShau/gotdbot"
)

func url(text, link string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text:  text,
		URL:   link,
		Style: style,
	}
}

func cb(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text:  text,
		Data:  []byte(data), // Agar gotdbot mein string field hai to `Data: data` kar dein
		Style: style,
	}
}

// URLButton builds a URL inline button.
func URLButton(text, link string) gotdbot.InlineKeyboardButton {
	return url(text, link, gotdbot.ButtonStylePrimary{})
}

// CBButton builds a callback inline button with the given style.
func CBButton(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return cb(text, data, style)
}

func randomStyles() [3]gotdbot.ButtonStyle {
	st := [3]gotdbot.ButtonStyle{
		gotdbot.ButtonStylePrimary{},
		gotdbot.ButtonStyleSuccess{},
		gotdbot.ButtonStyleDanger{},
	}
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

// PanelButtons is the now-playing keyboard.
// Layout: progress / Replay, Pause|Resume, Skip / Queue · N, End Queue / Autoplay.
// queue is the number of tracks waiting after the current one.
func PanelButtons(mode string, played, total int32, queue int, autoplay bool) *gotdbot.ReplyMarkupInlineKeyboard {
	progress := cb(ProgressBar(played, total), "play_progress", gotdbot.ButtonStyleDefault{})

	replay := cb("↺ Replay", "play_replay", gotdbot.ButtonStyleSuccess{})
	toggle := cb("II Pause", "play_pause", gotdbot.ButtonStylePrimary{})
	if mode == "pause" {
		toggle = cb("▷ Resume", "play_resume", gotdbot.ButtonStylePrimary{})
	}
	skip := cb("» Skip", "play_skip", gotdbot.ButtonStylePrimary{})

	if queue < 0 {
		queue = 0
	}
	queueBtn := cb(fmt.Sprintf("☰ Queue · %d", queue), "play_queue", gotdbot.ButtonStylePrimary{})
	endBtn := cb("▢ End Queue", "play_stop", gotdbot.ButtonStyleDanger{})

	apText := "Autoplay: Off"
	if autoplay {
		apText = "Autoplay: On"
	}
	ap := cb(apText, "play_autoplay", gotdbot.ButtonStyleDefault{})

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{progress},
			{replay, toggle, skip},
			{queueBtn, endBtn},
			{ap},
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

// panelHTML renders the card: header, track info, footer.
// header is plain small-caps text, extra an optional plain note.
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

// NowPlayingHTML builds the now-playing panel (rich message HTML).
// status is small-caps text; "now playing" style statuses show the
// "music is playing" header, anything else (paused, muted…) is shown as is.
// extra is an optional plain note.
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
