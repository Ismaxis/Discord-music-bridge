package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type telegramBot struct {
	config   config
	discord  *discordPlayer
	client   *http.Client
	playLock sync.Mutex
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text string `json:"text"`
}

type telegramResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func newTelegramBot(cfg config, discord *discordPlayer) *telegramBot {
	return &telegramBot{
		config:  cfg,
		discord: discord,
		client:  &http.Client{Timeout: 40 * time.Second},
	}
}

func (b *telegramBot) run(ctx context.Context) error {
	var offset int64
	for {
		updates, err := b.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("Telegram polling failed: %v", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				continue
			}
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *telegramBot) getUpdates(ctx context.Context, offset int64) ([]telegramUpdate, error) {
	payload, err := json.Marshal(map[string]any{
		"offset":  offset,
		"timeout": 30,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Telegram poll: %w", err)
	}
	body, err := b.call(ctx, "getUpdates", payload)
	if err != nil {
		return nil, err
	}
	var updates []telegramUpdate
	if err := json.Unmarshal(body, &updates); err != nil {
		return nil, fmt.Errorf("decode Telegram updates: %w", err)
	}
	return updates, nil
}

func (b *telegramBot) handleUpdate(ctx context.Context, update telegramUpdate) {
	if update.Message == nil || update.Message.Chat.ID != b.config.telegramChatID {
		return
	}
	text := strings.TrimSpace(update.Message.Text)
	switch text {
	case "/start", "/help":
		b.reply(ctx, "Send a YouTube URL to play its audio in the configured Discord voice channel.")
		return
	}
	rawURL := text
	if strings.HasPrefix(text, "/play ") {
		rawURL = strings.TrimSpace(strings.TrimPrefix(text, "/play "))
	}
	if !isYouTubeURL(rawURL) {
		b.reply(ctx, "Send a YouTube URL, or use /play followed by a YouTube URL.")
		return
	}
	if !b.playLock.TryLock() {
		b.reply(ctx, "Audio is already downloading or playing. Please wait for it to finish.")
		return
	}
	if err := b.sendMessage(ctx, "Downloading and preparing audio..."); err != nil {
		b.playLock.Unlock()
		log.Printf("Telegram status message failed: %v", err)
		return
	}
	go b.play(ctx, rawURL)
}

func (b *telegramBot) play(ctx context.Context, rawURL string) {
	defer b.playLock.Unlock()
	path, cleanup, err := downloadAudio(ctx, rawURL, b.config)
	if err != nil {
		b.reportError(ctx, err)
		return
	}
	defer cleanup()

	if err := b.discord.play(ctx, path, b.config.ffmpegPath); err != nil {
		b.reportError(ctx, err)
		return
	}
	if err := b.sendMessage(ctx, "Playback finished."); err != nil {
		log.Printf("Telegram completion message failed: %v", err)
	}
}

func (b *telegramBot) reportError(ctx context.Context, err error) {
	log.Printf("audio request failed: %v", err)
	if sendErr := b.sendMessage(ctx, "Could not play that audio: "+err.Error()); sendErr != nil {
		log.Printf("Telegram error message failed: %v", sendErr)
	}
}

func (b *telegramBot) reply(ctx context.Context, text string) {
	if err := b.sendMessage(ctx, text); err != nil {
		log.Printf("Telegram reply failed: %v", err)
	}
}

func (b *telegramBot) sendMessage(ctx context.Context, text string) error {
	payload, err := json.Marshal(map[string]string{
		"chat_id": strconv.FormatInt(b.config.telegramChatID, 10),
		"text":    text,
	})
	if err != nil {
		return fmt.Errorf("encode Telegram message: %w", err)
	}
	_, err = b.call(ctx, "sendMessage", payload)
	return err
}

func (b *telegramBot) call(ctx context.Context, method string, payload []byte) (json.RawMessage, error) {
	endpoint := "https://api.telegram.org/bot" + url.PathEscape(b.config.telegramToken) + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Telegram %s: %w", method, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read Telegram %s response: %w", method, err)
	}
	var result telegramResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode Telegram %s response: %w", method, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !result.OK {
		return nil, fmt.Errorf("Telegram %s failed (HTTP %d): %s", method, resp.StatusCode, result.Description)
	}
	return result.Result, nil
}
