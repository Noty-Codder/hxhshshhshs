package messaging

import (
	"fmt"
	"html"

	"istkharmusic/internal/cache"
	"istkharmusic/internal/utils"

	td "github.com/AshokShau/gotdbot"
)

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
		return msg, nil
	}

	plain := fmt.Sprintf("<b>%s</b>\n\n<b>%s:</b> <a href='%s'>%s</a>\n<b>%s:</b> %s\n<b>%s:</b> %s",
		status, utils.SC("track"), html.EscapeString(song.URL), html.EscapeString(song.Name),
		utils.SC("duration"), utils.SecToMin(song.Duration), utils.SC("requester"), html.EscapeString(song.User))
	return anchor.EditText(c, plain, &td.EditMessageTextOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           utils.ControlButtons("play"),
		DisableWebPagePreview: true,
	})
}