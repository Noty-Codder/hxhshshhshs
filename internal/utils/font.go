package utils

import "strings"

var smallCapsLetters = []string{
	"ᴧ", "ʙ", "ᴄ", "ᴅ", "є", "ғ", "ɢ", "ʜ", "ɪ", "ᴊ", "ᴋ", "ʟ", "ᴍ",
	"η", "σ", "ᴘ", "ǫ", "ʀ", "s", "ᴛ", "ᴜ", "ᴠ", "ᴡ", "x", "ʏ", "ᴢ",
}

// SC converts plain text to the small-caps font used across the bot.
// Only pass plain labels (never HTML markup).
func SC(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteString(smallCapsLetters[r-'a'])
		case r >= 'A' && r <= 'Z':
			b.WriteString(smallCapsLetters[r-'A'])
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
