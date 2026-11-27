package messaging

import (
	"fmt"
	"strings"
)

func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func RenderPlayPanelText(title, url, duration, requestedBy string) string {
	safeTitle := escapeHTML(title)
	safeURL := escapeHTML(url)
	safeBy := escapeHTML(requestedBy)

	return fmt.Sprintf(`<blockquote>🎧 <b>ᴍᴜꜱɪᴄ ɪꜱ ᴘʟᴀʏɪɴɢ</b>

🎵 <b>ɴᴏᴡ ᴘʟᴀʏɪɴɢ :</b> <a href="%s">%s</a>
⏱️ <b>ᴛʀᴀᴄᴋ ᴅᴜʀᴀᴛɪᴏɴ :</b> %s ᴍɪɴ
👤 <b>ʀᴇǫᴜᴇꜱᴛᴇᴅ ʙʏ :</b> %s

🎶 ʏᴏᴜʀ ᴛʀᴀᴄᴋ ɪꜱ ɴᴏᴡ ᴘʟᴀʏɪɴɢ
🔊 ꜱɪᴛ ʙᴀᴄᴋ, ʀᴇʟᴀx & ᴇɴᴊᴏʏ ᴛʜᴇ ᴍᴜꜱɪᴄ</blockquote>`, safeURL, safeTitle, duration, safeBy)
}
