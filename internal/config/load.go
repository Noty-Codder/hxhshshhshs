/*
  * IstkharMusic - Telegram Music Bot (by TEAM-ISTKHAR)
 * Copyright (c) 2025-2026 Ashok Shau
 *
 * Licensed under GNU GPL v3
 * See https://github.com/TEAM-ISTKHAR/IstkharMusic
 */

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

const (
	defaultDBName         = "anon"
	defaultAPIURL         = "https://www.riteshyt.in"
	defaultService        = "youtube"
	defaultDownloadsDir   = "downloads"
	defaultSupportGroup   = "https://t.me/betabot_support"
	defaultSupportChannel = "https://t.me/betabot_hub"
	defaultStartImage     = "https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg"
	defaultMaxFileSize    = int64(500 * 1024 * 1024)
	defaultSongDuration   = int32(3600)
	defaultSessionType    = "pyrogram"
	defaultMaxSessions    = 10
	defaultAutoPlay       = int32(10)
	defaultAutoLeave      = false
	defaultVideoPlayback  = true
	defaultAudioBitrate   = int64(128000)
)

var (
	ApiId          = getEnvInt32("API_ID", 0)
	ApiHash        = os.Getenv("API_HASH")
	Token          = os.Getenv("TOKEN")
	DlBotToken     = os.Getenv("DL_BOT_TOKEN")
	SessionStrings = getSessionStrings("STRING", defaultMaxSessions)
	SessionType    = getEnv("SESSION_TYPE", defaultSessionType)

	MongoUri = os.Getenv("MONGO_URI")
	DbName   = getEnv("DB_NAME", defaultDBName)

	ApiUrl = getEnv("API_URL", defaultAPIURL)
	ApiKey = getEnv("API_KEY", "")

	OwnerId  = getEnvInt64("OWNER_ID", 0)
	LoggerId = getEnvInt64("LOGGER_ID", 0)

	Proxy             = os.Getenv("PROXY")
	DefaultService    = strings.ToLower(getEnv("DEFAULT_SERVICE", defaultService))
	AutoPlayLimit     = getEnvInt32("AUTO_PLAY_LIMIT", defaultAutoPlay)
	MaxFileSize       = getEnvInt64("MAX_FILE_SIZE", defaultMaxFileSize)
	SongDurationLimit = getEnvInt32("SONG_DURATION_LIMIT", defaultSongDuration)
	AudioBitrate      = getEnvInt64("AUDIO_BITRATE", defaultAudioBitrate)
	DownloadsDir      = getEnv("DOWNLOADS_DIR", defaultDownloadsDir)

	SupportGroup   = getEnv("SUPPORT_GROUP", defaultSupportGroup)
	SupportChannel = getEnv("SUPPORT_CHANNEL", defaultSupportChannel)
	StartImg       = getEnv("START_IMG", "")
	PingImg        = getEnv("PING_IMG", "")
	SupportImg     = getEnv("SUPPORT_IMG", "")
	PlayImg        = getEnv("PLAY_IMG", "")
	WelcomeImg     = getEnv("WELCOME_IMG", "")
	BotName        = getEnv("BOT_NAME", "IstkharMusic")
	Port           = getEnv("PORT", "8080")
	BaseURL        = detectBaseURL()
	PromoHour      = int(getEnvInt32("PROMO_HOUR", 7))
	PromoEnabled   = getEnvBool("AUTO_PROMO", true)

	AutoLeave           = getEnvBool("AUTO_LEAVE", defaultAutoLeave)
	EnableVideoPlayback = getEnvBool("ENABLE_VPLAY", defaultVideoPlayback)

	DEVS        []int64
)

func LoadEnv() error {
	loadDevelopers()

	if OwnerId != 0 && !slices.Contains(DEVS, OwnerId) {
		DEVS = append(DEVS, OwnerId)
	}

	if err := validate(); err != nil {
		return fmt.Errorf("config is invalid: %w", err)
	}

	if err := os.MkdirAll(DownloadsDir, 0o755); err != nil {
		return errors.New("failed to create downloads directory")
	}

	return nil
}

func validate() error {
	required := []struct {
		name  string
		valid bool
	}{
		{"API_ID", ApiId > 0},
		{"API_HASH", ApiHash != ""},
		{"TOKEN", Token != ""},
		{"MONGO_URI", MongoUri != ""},
		{"OWNER_ID", OwnerId > 0},
	}

	var missing []string

	for _, field := range required {
		if !field.valid {
			missing = append(missing, field.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"missing required configuration: %s",
			strings.Join(missing, ", "),
		)
	}

	if len(SessionStrings) == 0 {
		return fmt.Errorf(
			"at least one session string is required (STRING or STRING1-%d)",
			defaultMaxSessions,
		)
	}

	if MaxFileSize <= 0 {
		return fmt.Errorf("MAX_FILE_SIZE must be greater than 0")
	}

	if SongDurationLimit <= 0 {
		return fmt.Errorf("SONG_DURATION_LIMIT must be greater than 0")
	}
	if AudioBitrate < 6000 || AudioBitrate > 510000 {
		return fmt.Errorf("AUDIO_BITRATE must be between 6000 and 510000 bits per second")
	}

	if !isValidService(DefaultService) {
		slog.Warn("invalid default service, falling back to default",
			"service", DefaultService,
			"default", defaultService,
		)

		DefaultService = defaultService
	}

	if AutoPlayLimit < 0 {
		AutoPlayLimit = 5
	}

	return nil
}

func isValidService(service string) bool {
	switch strings.ToLower(strings.TrimSpace(service)) {
	case "youtube", "spotify":
		return true
	default:
		return false
	}
}

func detectBaseURL() string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/"); v != "" {
		return v
	}
	if v := os.Getenv("HEROKU_APP_NAME"); v != "" {
		return "https://" + v + ".herokuapp.com"
	}
	if v := os.Getenv("RAILWAY_PUBLIC_DOMAIN"); v != "" {
		return "https://" + v
	}
	if v := strings.TrimRight(os.Getenv("RENDER_EXTERNAL_URL"), "/"); v != "" {
		return v
	}
	return ""
}

// Img resolves an image URL: the custom value wins, then the bundled banner
// served by this bot (needs BASE_URL or a platform URL), then the old default.
func Img(custom string) string {
	if custom != "" {
		return custom
	}
	if StartImg != "" && custom == "" {
		return StartImg
	}
	if BaseURL != "" {
		return BaseURL + "/banner.jpg"
	}
	return defaultStartImage
}

// ImgOr is like Img but prefers `custom`, then `fallback`, then the banner.
func ImgOr(custom, fallback string) string {
	if custom != "" {
		return custom
	}
	if fallback != "" {
		return fallback
	}
	return Img("")
}
