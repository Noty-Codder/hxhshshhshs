package utils

import (
	"fmt"
	"html"
	"math/rand/v2"
	"strings"

	"istkharmusic/internal/config"

	"github.com/AshokShau/gotdbot"
)

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

// ProgressBar renders "1:00 ───●───── -1:03".
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
	rem := total - played
	if rem < 0 {
		rem = 0
	}
	return fmt.Sprintf("%s %s -%s", SecToMin(played), b.String(), SecToMin(rem))
}

// PanelButtons is the Kanha-style now-playing keyboard (random colours each render).
// Layout: progress / ▷ II ⟳ ‣‣I ▢ / -15s 15s+ / autoplay / playlist + close.
func PanelButtons(mode string, played, total int32, autoplay bool) *gotdbot.ReplyMarkupInlineKeyboard {
	st := randomStyles()
	progress := cb(ProgressBar(played, total), "play_progress", gotdbot.ButtonStyleDefault{})

	resume := cb("▷", "play_resume", st[0])
	pause := cb("II", "play_pause", st[1])
	replay := cb("⟳", "play_replay", st[2])
	skip := cb("‣‣I", "play_skip", st[0])
	end := cb("▢", "play_stop", st[1])

	back := cb("-15s", "play_seek_back", st[2])
	fwd := cb("15s+", "play_seek_fwd", st[0])

	apText := "♫ " + SC("autoplay: off")
	apStyle := gotdbot.ButtonStyle(gotdbot.ButtonStyleDanger{})
	if autoplay {
		apText = "♫ " + SC("autoplay: on")
		apStyle = gotdbot.ButtonStyleSuccess{}
	}
	ap := cb(apText, "play_autoplay", apStyle)
	addPl := cb("➕ "+SC("playlist"), "play_add_to_list", gotdbot.ButtonStylePrimary{})

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{progress},
			{resume, pause, replay, skip, end},
			{back, fwd},
			{ap},
			{addPl, CloseBtn},
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

// NowPlayingHTML builds the premium panel (rich message HTML).
// status must already be small-caps text; extra is an optional plain note.
func NowPlayingHTML(song *PlayerCache, status, extra string) string {
	title := html.EscapeString(shorten(song.Name, 60))
	if song.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(song.URL), title)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s"/>`, config.ImgOr(config.PlayImg, ""))
	b.WriteString("<blockquote>")
	fmt.Fprintf(&b, "<b>➜ %s</b><br>", status)
	fmt.Fprintf(&b, "▶️ <b>%s</b> : %s<br>", SC("track"), title)
	fmt.Fprintf(&b, "⏱ <b>%s</b> : %s<br>", SC("duration"), SecToMin(song.Duration))
	fmt.Fprintf(&b, "👤 <b>%s</b> : %s", SC("requester"), html.EscapeString(song.User))
	if extra != "" {
		fmt.Fprintf(&b, "<br>✨ %s", html.EscapeString(extra))
	}
	b.WriteString("</blockquote>")
	return b.String()
}
