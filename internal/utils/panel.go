package utils

import (
	"fmt"
	"html"
	"math/rand/v2"
	"strings"

	"istkharmusic/internal/config"

	"github.com/AshokShau/gotdbot"
)

func URLButton(text, link string) gotdbot.InlineKeyboardButton {
	return url(text, link, gotdbot.ButtonStylePrimary{})
}

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

// ProgressBar renders compact "1:00 ──●──── -1:03".
func ProgressBar(played, total int32) string {
	const slots = 7
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
	rem := total - played
	if rem < 0 {
		rem = 0
	}
	return fmt.Sprintf("%s %s -%s", SecToMin(played), b.String(), SecToMin(rem))
}

// PanelButtons compact 3-row layout:
// Row 1: Progress Bar
// Row 2: -15s | ▷ | II | ‣‣I | ▢ | 15s+
// Row 3: Autoplay | +Playlist | Close
func PanelButtons(mode string, played, total int32, autoplay bool) *gotdbot.ReplyMarkupInlineKeyboard {
	st := randomStyles()
	progress := cb(ProgressBar(played, total), "play_progress", gotdbot.ButtonStyleDefault{})

	back := cb("-15s", "play_seek_back", st[2])
	resume := cb("▷", "play_resume", st[0])
	pause := cb("II", "play_pause", st[1])
	skip := cb("‣‣I", "play_skip", st[0])
	end := cb("▢", "play_stop", st[1])
	fwd := cb("15s+", "play_seek_fwd", st[2])

	apText := "♫ " + SC("ap: off")
	apStyle := gotdbot.ButtonStyle(gotdbot.ButtonStyleDanger{})
	if autoplay {
		apText = "♫ " + SC("ap: on")
		apStyle = gotdbot.ButtonStyleSuccess{}
	}
	ap := cb(apText, "play_autoplay", apStyle)
	addPl := cb("➕ "+SC("list"), "play_add_to_list", gotdbot.ButtonStylePrimary{})

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{progress},
			{back, resume, pause, skip, end, fwd},
			{ap, addPl, CloseBtn},
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

// NowPlayingHTML compact rich panel.
func NowPlayingHTML(song *PlayerCache, status, extra string) string {
	title := html.EscapeString(shorten(song.Name, 45))
	if song.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(song.URL), title)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s"/>`, config.ImgOr(config.PlayImg, ""))
	b.WriteString("<blockquote>")
	fmt.Fprintf(&b, "<b>➜ %s</b><br>", status)
	fmt.Fprintf(&b, "▶️ <b>%s:</b> %s<br>", SC("track"), title)
	fmt.Fprintf(&b, "⏱ <b>%s:</b> %s • 👤 <b>%s:</b> %s",
		SC("dur"), SecToMin(song.Duration),
		SC("by"), html.EscapeString(shorten(song.User, 18)))
	if extra != "" {
		fmt.Fprintf(&b, "<br>✨ %s", html.EscapeString(extra))
	}
	b.WriteString("</blockquote>")
	return b.String()
}
