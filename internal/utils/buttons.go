package utils

import (
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

type InlineButtonSpec struct {
	Text         string
	CallbackData string
	Style        string
}

func PlayPanelButtons(chatID int64, played, total, bar string, autoplayOn bool) [][]InlineButtonSpec {
	autoplayText := "ᴀᴜᴛᴏᴘʟᴀʏ: ᴏꜰꜰ"
	if autoplayOn {
		autoplayText = "ᴀᴜᴛᴏᴘʟᴀʏ: ᴏɴ"
	}

	timerLabel := fmt.Sprintf("%s %s %s", played, bar, total)
	if bar == "" {
		timerLabel = fmt.Sprintf("%s / %s", played, total)
	}

	return [][]InlineButtonSpec{
		{
			{
				Text:         timerLabel,
				CallbackData: "get_timer",
				Style:        "primary",
			},
		},
		{
			{
				Text:         "↺ ʀᴇᴘʟᴀʏ",
				CallbackData: fmt.Sprintf("admin_replay:%d", chatID),
				Style:        "success",
			},
			{
				Text:         "II ᴘᴀᴜꜱᴇ",
				CallbackData: fmt.Sprintf("admin_pause:%d", chatID),
				Style:        "primary",
			},
			{
				Text:         "» ꜱᴋɪᴘ",
				CallbackData: fmt.Sprintf("admin_skip:%d", chatID),
				Style:        "primary",
			},
		},
		{
			{
				Text:         "✚ ᴘʟᴀʏʟɪꜱᴛ",
				CallbackData: fmt.Sprintf("add_playlist:%d", chatID),
				Style:        "primary",
			},
			{
				Text:         "▢ ᴇɴᴅ ǫᴜᴇᴜᴇ",
				CallbackData: fmt.Sprintf("admin_stop:%d", chatID),
				Style:        "danger",
			},
		},
		{
			{
				Text:         autoplayText,
				CallbackData: fmt.Sprintf("admin_autoplay:%d", chatID),
				Style:        "",
			},
		},
	}
}

func ToGotdbotKeyboard(rows [][]InlineButtonSpec) *td.ReplyMarkupInlineKeyboard {
	kbRows := make([][]td.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		kbRow := make([]td.InlineKeyboardButton, 0, len(row))
		for _, btn := range row {
			kbRow = append(kbRow, td.InlineKeyboardButton{
				Text: btn.Text,
				Type: &td.InlineKeyboardButtonTypeCallback{
					Data: []byte(btn.CallbackData),
				},
			})
		}
		kbRows = append(kbRows, kbRow)
	}
	return &td.ReplyMarkupInlineKeyboard{Rows: kbRows}
}
