package bot

import (
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"istkharmusic/internal/config"
	"istkharmusic/internal/db"
	"istkharmusic/internal/messaging"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

var promoRunning atomic.Bool

func promoHTML() string {
	return `<img src="` + config.Img(config.StartImg) + `"/>` +
		"<h3>💫 " + utils.SC("turn your telegram group into a music zone.") + "</h3>" +
		"<blockquote>👋 " + utils.SC("add "+config.BotName+" to your group") + "<br>" +
		"🎵 " + utils.SC("play your favourite songs anytime") + "<br>" +
		"⚡ " + utils.SC("get fast search and playback") + "<br>" +
		"🎧 " + utils.SC("enjoy smooth music in voice chat") + "</blockquote>" +
		"<p>" + utils.SC("send /play song name and let the music begin.") + "</p>"
}

func promoMarkup(c *td.Client) *td.ReplyMarkupInlineKeyboard {
	return utils.AddMeMarkup(c.Me().Usernames.EditableUsername)
}

// runPromo sends the self-promo post to every served chat.
func runPromo(c *td.Client, report func(string)) {
	if !promoRunning.CompareAndSwap(false, true) {
		if report != nil {
			report("A promo run is already in progress.")
		}
		return
	}
	defer promoRunning.Store(false)

	chats, _ := db.Instance.GetAllChats()
	text, markup := promoHTML(), promoMarkup(c)
	ok, fail := 0, 0
	for _, id := range chats {
		if db.Instance.IsBlacklistedChat(id) {
			continue
		}
		if _, err := messaging.SendRich(c, id, text, markup); err != nil {
			if w := getFloodWait(err); w > 0 {
				time.Sleep(time.Duration(w+5) * time.Second)
			}
			fail++
		} else {
			ok++
		}
		time.Sleep(400 * time.Millisecond)
	}
	slog.Info("[promo] finished", "sent", ok, "failed", fail)
	if report != nil {
		report("✅ Promo finished.\nSent: " + strconv.Itoa(ok) + "\nFailed: " + strconv.Itoa(fail))
	}
}

// StartPromoScheduler posts the promo every day at PROMO_HOUR (IST, default 7 AM).
func StartPromoScheduler(c *td.Client) {
	if !config.PromoEnabled {
		return
	}
	go func() {
		for {
			now := time.Now().In(istZone)
			next := time.Date(now.Year(), now.Month(), now.Day(), config.PromoHour, 0, 0, 0, istZone)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			runPromo(c, nil)
		}
	}()
}

func selfPromoHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}
	if !strings.EqualFold(strings.TrimSpace(Args(m)), "run") {
		_, _ = m.ReplyText(c, "Usage: /selfpromo run", nil)
		return td.EndGroups
	}
	_, _ = m.ReplyText(c, "Promo started…", nil)
	go runPromo(c, func(s string) { _, _ = m.ReplyText(c, s, nil) })
	return td.EndGroups
}

// plainTriggerWatcher lets owners type "selfpromo run" without a slash.
func plainTriggerWatcher(c *td.Client, u *td.UpdateNewMessage) error {
	m := u.Message
	if m == nil {
		return td.ContinueGroups
	}
	if strings.EqualFold(strings.TrimSpace(m.GetText()), "selfpromo run") && isDev(c, m) {
		_, _ = m.ReplyText(c, "Promo started…", nil)
		go runPromo(c, func(s string) { _, _ = m.ReplyText(c, s, nil) })
	}
	return td.ContinueGroups
}
