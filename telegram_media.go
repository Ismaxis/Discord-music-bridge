package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const maxTelegramDownloadSize = 20 << 20

var audioFileExtensions = map[string]struct{}{
	".aac": {}, ".aif": {}, ".aiff": {}, ".alac": {}, ".amr": {},
	".ape": {}, ".au": {}, ".caf": {}, ".flac": {}, ".m4a": {},
	".m4b": {}, ".mp3": {}, ".mpga": {}, ".oga": {}, ".ogg": {},
	".opus": {}, ".wav": {}, ".weba": {}, ".wma": {},
}

func isAudioDocument(fileName, contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(strings.ToLower(mediaType), "audio/") {
		return true
	}
	_, ok := audioFileExtensions[strings.ToLower(filepath.Ext(fileName))]
	return ok
}

func (b *telegramBot) downloadTelegramAudio(ctx context.Context, file telegramFile) (string, func(), error) {
	if file.FileID == "" {
		return "", nil, fmt.Errorf("Telegram audio attachment is missing its file ID")
	}
	if file.FileSize > maxTelegramDownloadSize {
		return "", nil, fmt.Errorf("Telegram audio files must be no larger than 20 MiB")
	}

	payload, err := json.Marshal(map[string]string{"file_id": file.FileID})
	if err != nil {
		return "", nil, fmt.Errorf("encode Telegram file request: %w", err)
	}
	body, err := b.call(ctx, "getFile", payload)
	if err != nil {
		return "", nil, err
	}
	var fileInfo telegramFileInfo
	if err := json.Unmarshal(body, &fileInfo); err != nil {
		return "", nil, fmt.Errorf("decode Telegram file information: %w", err)
	}
	if fileInfo.FilePath == "" {
		return "", nil, fmt.Errorf("Telegram did not return a file path")
	}
	if fileInfo.FileSize > maxTelegramDownloadSize {
		return "", nil, fmt.Errorf("Telegram audio files must be no larger than 20 MiB")
	}

	fileURL, err := telegramFileURL(b.config.telegramToken, fileInfo.FilePath)
	if err != nil {
		return "", nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("create Telegram file download request: %w", err)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("download Telegram audio file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("Telegram file download failed (HTTP %d)", resp.StatusCode)
	}
	if resp.ContentLength > maxTelegramDownloadSize {
		return "", nil, fmt.Errorf("Telegram audio files must be no larger than 20 MiB")
	}

	dir, err := os.MkdirTemp("", "discord-music-telegram-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary audio directory: %w", err)
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("remove temporary Telegram audio directory: %v", err)
		}
	}
	audioPath := filepath.Join(dir, "upload")
	output, err := os.OpenFile(audioPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("create temporary Telegram audio file: %w", err)
	}

	written, copyErr := io.Copy(output, io.LimitReader(resp.Body, maxTelegramDownloadSize+1))
	closeErr := output.Close()
	if copyErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("save Telegram audio file: %w", copyErr)
	}
	if closeErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temporary Telegram audio file: %w", closeErr)
	}
	if written > maxTelegramDownloadSize {
		cleanup()
		return "", nil, fmt.Errorf("Telegram audio files must be no larger than 20 MiB")
	}
	if written == 0 {
		cleanup()
		return "", nil, fmt.Errorf("Telegram audio file is empty")
	}
	return audioPath, cleanup, nil
}

func telegramFileURL(token, filePath string) (string, error) {
	if token == "" || strings.HasPrefix(filePath, "/") || strings.Contains(filePath, `\`) {
		return "", fmt.Errorf("invalid Telegram file location")
	}
	parts := strings.Split(filePath, "/")
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid Telegram file location")
		}
		parts[i] = url.PathEscape(part)
	}
	return "https://api.telegram.org/file/bot" +
		url.PathEscape(token) + "/" + path.Join(parts...), nil
}
