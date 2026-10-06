package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type config struct {
	discordToken          string
	discordGuildID        string
	voiceChannelID        string
	telegramToken         string
	telegramChatID        int64
	youtubeDLPath         string
	youtubeCookiesFile    string
	youtubeCookiesBrowser string
	youtubeJSRuntime      string
	ffmpegPath            string
}

func loadConfig() (config, error) {
	var cfg config
	var err error
	if cfg.discordToken, err = requiredEnv("DISCORD_TOKEN"); err != nil {
		return cfg, err
	}
	if cfg.discordGuildID, err = requiredEnv("DISCORD_GUILD_ID"); err != nil {
		return cfg, err
	}
	guildID, err := strconv.ParseUint(cfg.discordGuildID, 10, 64)
	if err != nil || guildID == 0 {
		return cfg, fmt.Errorf("DISCORD_GUILD_ID must be a positive integer")
	}
	if cfg.voiceChannelID, err = requiredEnv("DISCORD_VOICE_CHANNEL_ID"); err != nil {
		return cfg, err
	}
	channelID, err := strconv.ParseUint(cfg.voiceChannelID, 10, 64)
	if err != nil || channelID == 0 {
		return cfg, fmt.Errorf("DISCORD_VOICE_CHANNEL_ID must be a positive integer")
	}
	if cfg.telegramToken, err = requiredEnv("TELEGRAM_BOT_TOKEN"); err != nil {
		return cfg, err
	}
	chatID, err := requiredEnv("TELEGRAM_CHAT_ID")
	if err != nil {
		return cfg, err
	}
	if cfg.telegramChatID, err = strconv.ParseInt(chatID, 10, 64); err != nil || cfg.telegramChatID == 0 {
		return cfg, fmt.Errorf("TELEGRAM_CHAT_ID must be a non-zero integer")
	}
	if cfg.youtubeDLPath, err = optionalEnv("YTDLP_PATH", "yt-dlp"); err != nil {
		return cfg, err
	}
	if cfg.youtubeCookiesFile, err = optionalEnv("YTDLP_COOKIES_FILE", ""); err != nil {
		return cfg, err
	}
	if cfg.youtubeCookiesBrowser, err = optionalEnv("YTDLP_COOKIES_FROM_BROWSER", ""); err != nil {
		return cfg, err
	}
	if cfg.youtubeCookiesFile != "" && cfg.youtubeCookiesBrowser != "" {
		return cfg, fmt.Errorf("set only one of YTDLP_COOKIES_FILE and YTDLP_COOKIES_FROM_BROWSER")
	}
	if cfg.youtubeJSRuntime, err = optionalEnv("YTDLP_JS_RUNTIME", ""); err != nil {
		return cfg, err
	}
	if cfg.ffmpegPath, err = optionalEnv("FFMPEG_PATH", "ffmpeg"); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

func optionalEnv(name, fallback string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	return value, nil
}
