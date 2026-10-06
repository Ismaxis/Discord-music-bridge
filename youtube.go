package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func isYouTubeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" && u.Scheme != "http" || u.User != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be", "www.youtu.be":
		return true
	default:
		return false
	}
}

func downloadAudio(ctx context.Context, rawURL string, cfg config) (string, func(), error) {
	if !isYouTubeURL(rawURL) {
		return "", nil, fmt.Errorf("please send a valid YouTube URL")
	}
	dir, err := os.MkdirTemp("", "discord-music-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary audio directory: %w", err)
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("remove temporary audio directory: %v", err)
		}
	}
	output := filepath.Join(dir, "source.%(ext)s")
	cmd := exec.CommandContext(ctx, cfg.youtubeDLPath, youtubeDLArgs(rawURL, output, cfg)...)
	result, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		return "", nil, fmt.Errorf("yt-dlp failed: %w: %s", err, strings.TrimSpace(string(result)))
	}

	matches, err := filepath.Glob(filepath.Join(dir, "source.*"))
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("find downloaded audio: %w", err)
	}
	if len(matches) != 1 {
		cleanup()
		return "", nil, fmt.Errorf("yt-dlp produced %d audio files; expected exactly one", len(matches))
	}
	info, err := os.Stat(matches[0])
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("inspect downloaded audio: %w", err)
	}
	if !info.Mode().IsRegular() {
		cleanup()
		return "", nil, fmt.Errorf("yt-dlp output is not a regular file")
	}
	return matches[0], cleanup, nil
}

func youtubeDLArgs(rawURL, output string, cfg config) []string {
	args := []string{
		"--no-playlist", "--no-progress", "--max-filesize", "100M",
		"--format", "bestaudio", "--output", output,
	}
	switch {
	case cfg.youtubeCookiesFile != "":
		args = append(args, "--cookies", cfg.youtubeCookiesFile)
	case cfg.youtubeCookiesBrowser != "":
		args = append(args, "--cookies-from-browser", cfg.youtubeCookiesBrowser)
	}
	if cfg.youtubeJSRuntime != "" {
		args = append(args, "--js-runtimes", cfg.youtubeJSRuntime)
	}
	return append(args, rawURL)
}
