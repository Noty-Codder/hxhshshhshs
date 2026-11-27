package bot

import (
	"fmt"
	"runtime"
	"time"

	"istkharmusic/internal/cache"
	"istkharmusic/internal/db"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

var istZone = time.FixedZone("IST", 5*3600+1800)

// bdataHandler: daily group add/remove report (IST day + last 24h).
func bdataHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}
	now := time.Now().In(istZone)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, istZone)
	h24 := now.Add(-24 * time.Hour)

	addToday := db.Instance.CountChatEvents("add", dayStart)
	remToday := db.Instance.CountChatEvents("remove", dayStart)
	add24 := db.Instance.CountChatEvents("add", h24)
	rem24 := db.Instance.CountChatEvents("remove", h24)

	chats, _ := db.Instance.GetAllChats()
	users, _ := db.Instance.GetAllUsers()
	active := len(cache.ChatCache.GetActiveChats())

	text := fmt.Sprintf(`<b>📊 %s</b> <i>(%s)</i>

<blockquote><b>%s</b>
➕ %s : <b>%d</b>
➖ %s : <b>%d</b>
📈 %s : <b>%+d</b></blockquote>

<blockquote><b>%s</b>
➕ %s : <b>%d</b>
➖ %s : <b>%d</b>
📈 %s : <b>%+d</b></blockquote>

<blockquote>🏘 %s : <b>%d</b>
👤 %s : <b>%d</b>
🎧 %s : <b>%d</b></blockquote>`,
		utils.SC("bot data"), now.Format("02 Jan 2006, 03:04 PM"),
		utils.SC("today (since 12 am ist)"),
		utils.SC("groups added"), addToday, utils.SC("groups removed"), remToday, utils.SC("net growth"), addToday-remToday,
		utils.SC("last 24 hours"),
		utils.SC("groups added"), add24, utils.SC("groups removed"), rem24, utils.SC("net growth"), add24-rem24,
		utils.SC("total groups"), len(chats), utils.SC("total users"), len(users), utils.SC("active vcs"), active)

	_, err := m.ReplyText(c, text, &td.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// vcStatusHandler: /vc — active voice chat health overview.
func vcStatusHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}
	active := cache.ChatCache.GetActiveChats()
	seen := map[int64]bool{}
	audio, video, stuck, dup := 0, 0, 0, 0
	for _, id := range active {
		if seen[id] {
			dup++
			continue
		}
		seen[id] = true
		t := cache.ChatCache.GetPlayingTrack(id)
		switch {
		case t == nil:
			stuck++
		case t.IsVideo:
			video++
		default:
			audio++
		}
	}
	healthy := audio + video
	verdict := "✅ " + utils.SC("all active vcs are unique.")
	if dup > 0 {
		verdict = fmt.Sprintf("⚠️ %d %s", dup, utils.SC("duplicate vcs found."))
	}

	text := fmt.Sprintf(`<b>🎧 %s</b>

<blockquote>📊 %s : <b>%d</b>
🎵 %s : <b>%d</b>
🔊 %s : <b>%d</b>
🎥 %s : <b>%d</b>

🧩 %s : <b>%d</b>
⚠️ %s : <b>%d</b>

%s</blockquote>`,
		utils.SC("active voice chat status"),
		utils.SC("server's vcpu"), runtime.NumCPU(),
		utils.SC("total active"), len(seen),
		utils.SC("audio"), audio, utils.SC("video"), video,
		utils.SC("unique healthy chats"), healthy,
		utils.SC("vc stuck"), stuck, verdict)
	_, err := m.ReplyText(c, text, &td.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// blacklistHandler: /blacklist <id> — negative = chat, positive = user.
func blacklistHandler(add bool) func(*td.Client, *td.Message) error {
	return func(c *td.Client, m *td.Message) error {
		if !isDev(c, m) {
			return td.EndGroups
		}
		var id int64
		if _, err := fmt.Sscan(Args(m), &id); err != nil || id == 0 {
			if uid, e := getTargetUserID(c, m); e == nil && uid != 0 {
				id = uid
			} else {
				_, _ = m.ReplyText(c, "Usage: /blacklist <chat_id|user_id> (negative = chat)", nil)
				return td.EndGroups
			}
		}
		var err error
		switch {
		case id < 0 && add:
			err = db.Instance.AddBlacklistedChat(id)
		case id < 0:
			err = db.Instance.RemoveBlacklistedChat(id)
		case add:
			err = db.Instance.AddBlacklistedUser(id)
		default:
			err = db.Instance.RemoveBlacklistedUser(id)
		}
		if err != nil {
			_, _ = m.ReplyText(c, "Failed: "+err.Error(), nil)
			return td.EndGroups
		}
		verb := "blacklisted"
		if !add {
			verb = "removed from blacklist"
		}
		_, _ = m.ReplyText(c, fmt.Sprintf("✅ %d %s.", id, verb), nil)
		return td.EndGroups
	}
}
