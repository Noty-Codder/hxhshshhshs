package messaging

import (
	"fmt"
	"html"
	"sync"
	"time"

	"istkharmusic/internal/cache"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

// PlayedTimeFunc is wired from calls package to avoid circular imports.
var PlayedTimeFunc func(chatID int64) (uint64, error)

var (
	tickerMu     sync.Mutex
	activePanels = make(map[int64]int64) // chatID -> active panel messageID
)

// StartPanelTicker runs a live 5s updater for the active now-playing panel.
func StartPanelTicker(c *td.Client, chatID, msgID int64) {
	tickerMu.Lock()
	activePanels[chatID] = msgID
	tickerMu.Unlock()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			tickerMu.Lock()
			currentID, ok := activePanels[chatID]
			tickerMu.Unlock()
			if !ok || currentID != msgID {
				return
			}

			song := cache.ChatCache.GetPlayingTrack(chatID)
			if song == nil || song.Duration <= 0 {
				StopPanelTicker(chatID)
				return
			}

			var played int32
			if PlayedTimeFunc != nil {
				if p, err := PlayedTimeFunc(chatID); err == nil {
					played = int32(p)
				}
			}

			markup := utils.PanelButtons("play", played, song.Duration, cache.ChatCache.GetAutoplay(chatID))
			// Sirf ReplyMarkup edit karne se message text/image flicker nahi hota aur flood limit bhi bachti hai
			_, err := c.EditMessageReplyMarkup(chatID, msgID, &td.EditMessageReplyMarkupOpts{
				ReplyMarkup: markup,
			})
			if err != nil {
				// Agar message delete ho gaya ya modify nahi hua to ignore/stop
				continue
			}
		}
	}()
}

// StopPanelTicker stops live progress updates for a chat.
func StopPanelTicker(chatID int64) {
	tickerMu.Lock()
	delete(activePanels, chatID)
	tickerMu.Unlock()
}

// SendRich sends a rich (image + text + buttons) message to a chat.
func SendRich(c *td.Client, chatID int64, htmlText string, markup *td.ReplyMarkupInlineKeyboard) (*td.Message, error) {
	ph, err := c.SendMessage(chatID, "✨", nil)
	if err != nil {
		return nil, err
	}
	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: htmlText}}
	if _, err = c.EditRichMessage(chatID, ph.Id, rich, &td.EditRichMessageOpts{ReplyMarkup: markup}); err == nil {
		return ph, nil
	}
	msg, err2 := ph.ReplyRichMessage(c, rich, &td.SendMessageOpts{ReplyMarkup: markup})
	_ = ph.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
	if err2 != nil {
		return nil, err2
	}
	return msg, nil
}

// SendPanel replaces the progress message with the premium now-playing panel.
func SendPanel(c *td.Client, anchor *td.Message, song *utils.PlayerCache, explicitSilent bool) (*td.Message, error) {
	if anchor == nil || anchor.Id == 0 {
		return nil, nil
	}
	if Silent(anchor.ChatId, explicitSilent) {
		_ = anchor.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
		return nil, nil
	}

	status := utils.SC("streaming started | enjoy music")
	markup := utils.PanelButtons("play", 0, song.Duration, cache.ChatCache.GetAutoplay(anchor.ChatId))
	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: utils.NowPlayingHTML(song, status, "")}}

	msg, err := anchor.ReplyRichMessage(c, rich, &td.SendMessageOpts{ReplyMarkup: markup})
	if err == nil {
		_ = anchor.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
		StartPanelTicker(c, anchor.ChatId, msg.Id)
		return msg, nil
	}

	plain := fmt.Sprintf("<b>%s</b>\n\n<b>%s:</b> <a href='%s'>%s</a>\n<b>%s:</b> %s\n<b>%s:</b> %s",
		status, utils.SC("track"), html.EscapeString(song.URL), html.EscapeString(song.Name),
		utils.SC("duration"), utils.SecToMin(song.Duration), utils.SC("requester"), html.EscapeString(song.User))
	edited, editErr := anchor.EditText(c, plain, &td.EditMessageTextOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           markup,
		DisableWebPagePreview: true,
	})
	if editErr == nil && edited != nil {
		StartPanelTicker(c, anchor.ChatId, edited.Id)
	}
	return edited, editErr
}
