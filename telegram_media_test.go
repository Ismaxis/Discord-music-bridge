package main

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestTelegramMessageAudioFile(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "audio",
			json: `{"audio":{"file_id":"audio-id","file_size":123,"mime_type":"audio/mpeg"}}`,
			want: "audio-id",
		},
		{
			name: "voice",
			json: `{"voice":{"file_id":"voice-id","file_size":123,"mime_type":"audio/ogg"}}`,
			want: "voice-id",
		},
		{
			name: "audio document by extension",
			json: `{"document":{"file_id":"document-id","file_size":123,"mime_type":"application/octet-stream","file_name":"track.mp3"}}`,
			want: "document-id",
		},
		{
			name: "audio document by MIME type",
			json: `{"document":{"file_id":"document-id","file_size":123,"mime_type":"audio/flac","file_name":"track.bin"}}`,
			want: "document-id",
		},
		{
			name: "non-audio document",
			json: `{"document":{"file_id":"document-id","file_size":123,"mime_type":"application/pdf","file_name":"document.pdf"}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var message telegramMessage
			if err := json.Unmarshal([]byte(test.json), &message); err != nil {
				t.Fatalf("decode message: %v", err)
			}
			file, ok := message.audioFile()
			if test.want == "" {
				if ok {
					t.Fatalf("audioFile() unexpectedly returned %q", file.FileID)
				}
				return
			}
			if !ok || file.FileID != test.want {
				t.Fatalf("audioFile() = (%q, %t), want (%q, true)", file.FileID, ok, test.want)
			}
		})
	}
}

func TestTelegramFileURL(t *testing.T) {
	got, err := telegramFileURL("123:secret", "voice/file name.ogg")
	if err != nil {
		t.Fatalf("telegramFileURL returned error: %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse Telegram file URL: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "api.telegram.org" {
		t.Fatalf("unexpected Telegram file URL origin: %s", got)
	}
	if parsed.EscapedPath() != "/file/bot123:secret/voice/file%20name.ogg" {
		t.Fatalf("unexpected Telegram file URL path: %s", parsed.EscapedPath())
	}
}

func TestTelegramFileURLRejectsUnsafePath(t *testing.T) {
	for _, filePath := range []string{
		"",
		"/absolute/file.ogg",
		"../outside.ogg",
		"voice/../outside.ogg",
		`voice\file.ogg`,
	} {
		if _, err := telegramFileURL("123:secret", filePath); err == nil {
			t.Errorf("telegramFileURL accepted unsafe path %q", filePath)
		}
	}
}

func TestIsTelegramCommand(t *testing.T) {
	tests := []struct {
		text    string
		command string
		want    bool
	}{
		{"/stop", "stop", true},
		{"/STOP", "stop", true},
		{"/stop@music_bot", "stop", true},
		{"/stop now", "stop", false},
		{"/stopwatch", "stop", false},
		{"stop", "stop", false},
		{"/help", "stop", false},
	}
	for _, test := range tests {
		t.Run(test.text, func(t *testing.T) {
			if got := isTelegramCommand(test.text, test.command); got != test.want {
				t.Errorf("isTelegramCommand(%q, %q) = %t, want %t", test.text, test.command, got, test.want)
			}
		})
	}
}
