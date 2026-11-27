package utils

import "fmt"

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

	return [][]InlineButtonSpec{
		{
			{
				Text:         fmt.Sprintf("%s %s %s", played, bar, total),
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
