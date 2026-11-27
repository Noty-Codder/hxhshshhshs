package bot

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	"istkharmusic/internal/config"
	"istkharmusic/internal/db"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

var bioLinkRe = regexp.MustCompile(`(?i)(https?://|t\.me/|telegram\.me/|www\.|@\w{5,})`)

type bioEntry struct {
	bad bool
	at  time.Time
}

var (
	bioMu     sync.Mutex
	bioCache  = map[int64]bioEntry{}
	warnMu    sync.Mutex
	warnStamp = map[string]time.Time{}
)

func isDevID(id int64) bool {
	for _, d := range config.DEVS {
		if d == id {
			return true
		}
	}
	return false
}

func isChatAdmin(chatID, userID int64) bool {
	return isDevID(userID) || db.Instance.IsAdmin(chatID, userID) || db.Instance.IsAuthUser(chatID, userID)
}

func msgHasLink(m *td.Message) bool {
	for _, e := range m.GetEntities() {
		switch e.Type.(type) {
		case *td.TextEntityTypeUrl, *td.TextEntityTypeTextUrl:
			return true
		}
	}
	return false
}

func userBioHasLink(c *td.Client, userID int64) bool {
	bioMu.Lock()
	if e, ok := bioCache[userID]; ok && time.Since(e.at) < 30*time.Minute {
		bioMu.Unlock()
		return e.bad
	}
	bioMu.Unlock()

	bad := false
	if info, err := c.GetUserFullInfo(userID); err == nil && info != nil {
		if raw, err := json.Marshal(info); err == nil {
			var data map[string]any
			if json.Unmarshal(raw, &data) == nil {
				bio := ""
				switch v := data["bio"].(type) {
				case string:
					bio = v
				case map[string]any:
					bio, _ = v["text"].(string)
				}
				bad = bioLinkRe.MatchString(bio)
			}
		}
	}

	bioMu.Lock()
	bioCache[userID] = bioEntry{bad: bad, at: time.Now()}
	bioMu.Unlock()
	return bad
}

func restrictWatcher(c *td.Client, u *td.UpdateNewMessage) error {
	m := u.Message
	if m == nil || m.ChatId >= 0 {
		return td.ContinueGroups
	}
	uid := m.SenderID()
	if uid <= 0 || uid == c.Me().Id {
		return td.ContinueGroups
	}

	cfg := db.Instance.GetRestrict(m.ChatId)
	if !cfg.LinkRemove && !cfg.BioRemove {
		return td.ContinueGroups
	}
	for _, a := range cfg.Approved {
		if a == uid {
			return td.ContinueGroups
		}
	}
	if isChatAdmin(m.ChatId, uid) {
		return td.ContinueGroups
	}

	reason := ""
	if cfg.LinkRemove && !strings.HasPrefix(strings.TrimSpace(m.GetText()), "/") && msgHasLink(m) {
		reason = "link"
	}
	if reason == "" && cfg.BioRemove && userBioHasLink(c, uid) {
		reason = "bio"
	}
	if reason == "" {
		return td.ContinueGroups
	}

	_ = c.DeleteMessages(m.ChatId, []int64{m.Id}, &td.DeleteMessagesOpts{Revoke: true})

	key := fmt.Sprintf("%d:%d", m.ChatId, uid)
	warnMu.Lock()
	last := warnStamp[key]
	allow := time.Since(last) > 30*time.Second
	if allow {
		warnStamp[key] = time.Now()
	}
	warnMu.Unlock()
	if !allow {
		return td.ContinueGroups
	}

	name := "User"
	if usr, err := c.GetUser(uid); err == nil && usr != nil {
		name = usr.FirstName
	}
	what := "ʟɪɴᴋꜱ ᴀʀᴇ ɴᴏᴛ ᴀʟʟᴏᴡᴇᴅ ʜᴇʀᴇ."
	if reason == "bio" {
		what = "ʟɪɴᴋꜱ ɪɴ ʏᴏᴜʀ ʙɪᴏ ᴀʀᴇ ɴᴏᴛ ᴀʟʟᴏᴡᴇᴅ ʜᴇʀᴇ."
	}
	text := fmt.Sprintf("🚫 <a href=\"tg://user?id=%d\">%s</a>, %s", uid, html.EscapeString(name), what)
	if sent, err := c.SendMessage(m.ChatId, text, &td.SendMessageOpts{ParseMode: "HTML"}); err == nil && sent != nil {
		chatID, id := sent.ChatId, sent.Id
		time.AfterFunc(8*time.Second, func() {
			_ = c.DeleteMessages(chatID, []int64{id}, &td.DeleteMessagesOpts{Revoke: true})
		})
	}
	return td.ContinueGroups
}

func parseToggle(args string) (on bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(args)) {
	case "on", "enable", "yes", "true", "1":
		return true, true
	case "off", "disable", "no", "false", "0":
		return false, true
	}
	return false, false
}

func stateText(on bool) string {
	if on {
		return "✅ " + utils.SC("on")
	}
	return "❌ " + utils.SC("off")
}

func adminOnlyGroup(c *td.Client, m *td.Message) bool {
	if m.ChatId >= 0 {
		_, _ = m.ReplyText(c, "This command only works in groups.", nil)
		return false
	}
	if !isChatAdmin(m.ChatId, m.SenderID()) {
		_, _ = m.ReplyText(c, "You must be an administrator to use this command.", nil)
		return false
	}
	return true
}

func toggleHandler(label string, get func(*db.RestrictCfg) bool, set func(int64, bool) error) func(*td.Client, *td.Message) error {
	return func(c *td.Client, m *td.Message) error {
		if !adminOnlyGroup(c, m) {
			return td.EndGroups
		}
		on, ok := parseToggle(Args(m))
		if !ok {
			cur := get(db.Instance.GetRestrict(m.ChatId))
			_, _ = m.ReplyText(c, fmt.Sprintf("<b>%s</b>: %s\n\nUsage: <code>on</code> / <code>off</code>", utils.SC(label), stateText(cur)), &td.SendMessageOpts{ParseMode: "HTML"})
			return td.EndGroups
		}
		if err := set(m.ChatId, on); err != nil {
			_, _ = m.ReplyText(c, "Failed to save the setting.", nil)
			return td.EndGroups
		}
		_, _ = m.ReplyText(c, fmt.Sprintf("<b>%s</b>: %s", utils.SC(label), stateText(on)), &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
}

func linkRemoveHandler(c *td.Client, m *td.Message) error {
	return toggleHandler("link remover", func(r *db.RestrictCfg) bool { return r.LinkRemove }, db.Instance.SetLinkRemove)(c, m)
}

func bioRemoveHandler(c *td.Client, m *td.Message) error {
	return toggleHandler("bio remover", func(r *db.RestrictCfg) bool { return r.BioRemove }, db.Instance.SetBioRemove)(c, m)
}

func vcLoggerToggleHandler(c *td.Client, m *td.Message) error {
	return toggleHandler("vc logger", func(r *db.RestrictCfg) bool { return r.VcLogger }, db.Instance.SetVcLogger)(c, m)
}

func approveHandler(c *td.Client, m *td.Message) error {
	if !adminOnlyGroup(c, m) {
		return td.EndGroups
	}
	uid, err := getTargetUserID(c, m)
	if err != nil || uid == 0 {
		_, _ = m.ReplyText(c, "Reply to a user or pass a user ID: /approve <id>", nil)
		return td.EndGroups
	}
	if err := db.Instance.ApproveUser(m.ChatId, uid); err != nil {
		_, _ = m.ReplyText(c, "Failed to approve the user.", nil)
		return td.EndGroups
	}
	_, _ = m.ReplyText(c, fmt.Sprintf("✅ <code>%d</code> %s", uid, utils.SC("approved (exempt from link/bio remover)")), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func unapproveHandler(c *td.Client, m *td.Message) error {
	if !adminOnlyGroup(c, m) {
		return td.EndGroups
	}
	uid, err := getTargetUserID(c, m)
	if err != nil || uid == 0 {
		_, _ = m.ReplyText(c, "Reply to a user or pass a user ID: /unapprove <id>", nil)
		return td.EndGroups
	}
	if err := db.Instance.UnapproveUser(m.ChatId, uid); err != nil {
		_, _ = m.ReplyText(c, "Failed to unapprove the user.", nil)
		return td.EndGroups
	}
	_, _ = m.ReplyText(c, fmt.Sprintf("❌ <code>%d</code> %s", uid, utils.SC("unapproved")), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func approvedListHandler(c *td.Client, m *td.Message) error {
	if !adminOnlyGroup(c, m) {
		return td.EndGroups
	}
	cfg := db.Instance.GetRestrict(m.ChatId)
	if len(cfg.Approved) == 0 {
		_, _ = m.ReplyText(c, "No approved users in this chat.", nil)
		return td.EndGroups
	}
	var sb strings.Builder
	sb.WriteString("<b>" + utils.SC("approved users") + "</b>\n")
	for i, id := range cfg.Approved {
		sb.WriteString(fmt.Sprintf("%d. <code>%d</code>\n", i+1, id))
	}
	_, _ = m.ReplyText(c, sb.String(), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}
