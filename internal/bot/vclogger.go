package bot

import (
	"fmt"
	"html"
	"sync"
	"time"

	"istkharmusic/internal/cache"
	"istkharmusic/internal/db"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

type vcKey struct{ chat, user int64 }

var (
	vcMu        sync.Mutex
	vcJoinCount = map[vcKey]int{}
)

const vcLogDeleteDelay = 7 * time.Second

// vcParticipantEvent is called by the calls package when someone joins/leaves a VC.
func vcParticipantEvent(c *td.Client, chatID, userID int64, joined bool) {
	if userID <= 0 || userID == c.Me().Id || !db.Instance.GetRestrict(chatID).VcLogger {
		return
	}

	name, username, role := "User", "Ignored", "Member"
	if usr, err := c.GetUser(userID); err == nil && usr != nil {
		name = usr.FirstName
		if usr.LastName != "" {
			name += " " + usr.LastName
		}
		if usr.Usernames != nil && usr.Usernames.EditableUsername != "" {
			username = "@" + usr.Usernames.EditableUsername
		}
	}
	if db.Instance.IsAdmin(chatID, userID) {
		role = "Admin"
	}
	if _, err := cache.GetUserAdmin(c, chatID, userID, false); err == nil && role != "Admin" {
		role = "Admin"
	}

	tag, action := "#JoinVideoChat", "Joined ["+role+"]"
	if !joined {
		tag, action = "#LeaveVideoChat", "Left ["+role+"]"
	}
	text := fmt.Sprintf("<b>%s</b>\n\n<blockquote>Name ➜ <a href=\"tg://user?id=%d\">%s</a>\nId ➜ <code>%d</code>\nUsername ➜ %s\nAction ➜ %s</blockquote>",
		tag, userID, html.EscapeString(name), userID, html.EscapeString(username), action)

	if joined {
		vcMu.Lock()
		k := vcKey{chatID, userID}
		vcJoinCount[k]++
		text += fmt.Sprintf("\n🔄 <b>Join Count</b> ➜ <code>%d</code>", vcJoinCount[k])
		vcMu.Unlock()
	}

	opts := &td.SendMessageOpts{ParseMode: "HTML"}
	if sg, err := getSupergroup(c, chatID); err == nil && sg != nil && sg.Usernames != nil && sg.Usernames.EditableUsername != "" {
		opts.ReplyMarkup = &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{
			{utils.URLButton("Join Live Vc 📶", "https://t.me/"+sg.Usernames.EditableUsername+"?videochat")},
		}}
	}

	sent, err := c.SendMessage(chatID, text, opts)
	if err != nil || sent == nil {
		return
	}
	cid, mid := sent.ChatId, sent.Id
	time.AfterFunc(vcLogDeleteDelay, func() {
		_ = c.DeleteMessages(cid, []int64{mid}, &td.DeleteMessagesOpts{Revoke: true})
	})
}
