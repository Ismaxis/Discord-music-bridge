package main

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReadOpusPackets(t *testing.T) {
	input := append(
		makeOggPage([]byte("OpusHead"), []byte("OpusTags")),
		makeOggPage([]byte{0xF8, 0xFF, 0xFE}, []byte{0xF8, 0xFF, 0xFD})...,
	)
	var packets [][]byte
	err := readOpusPackets(bytes.NewReader(input), func(packet []byte) error {
		packets = append(packets, packet)
		return nil
	})
	if err != nil {
		t.Fatalf("readOpusPackets returned error: %v", err)
	}
	if len(packets) != 2 {
		t.Fatalf("got %d audio packets, want 2", len(packets))
	}
	if !bytes.Equal(packets[0], []byte{0xF8, 0xFF, 0xFE}) ||
		!bytes.Equal(packets[1], []byte{0xF8, 0xFF, 0xFD}) {
		t.Fatalf("unexpected audio packets: %v", packets)
	}
}

func TestReadOpusPacketsRejectsNonOpusStream(t *testing.T) {
	input := makeOggPage([]byte("Vorbis!!"))
	if err := readOpusPackets(bytes.NewReader(input), func([]byte) error { return nil }); err == nil {
		t.Fatal("expected an error for a non-Opus stream")
	}
}

func TestStreamOpusWithFFmpeg(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	inputPath := filepath.Join(t.TempDir(), "sample.wav")
	output, err := exec.Command(ffmpeg,
		"-hide_banner", "-loglevel", "error", "-f", "lavfi",
		"-i", "sine=frequency=440:duration=0.2", "-ar", "48000", "-ac", "2",
		"-y", inputPath,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("create sample audio with ffmpeg: %v: %s", err, output)
	}

	packetCount := 0
	err = streamOpus(context.Background(), inputPath, ffmpeg, func([]byte) error {
		packetCount++
		return nil
	})
	if err != nil {
		t.Fatalf("streamOpus returned error: %v", err)
	}
	if packetCount == 0 {
		t.Fatal("streamOpus produced no Opus packets")
	}
}

func makeOggPage(packets ...[]byte) []byte {
	var segments []byte
	var body []byte
	for _, packet := range packets {
		for len(packet) >= 255 {
			segments = append(segments, 255)
			body = append(body, packet[:255]...)
			packet = packet[255:]
		}
		segments = append(segments, byte(len(packet)))
		body = append(body, packet...)
	}

	page := make([]byte, 27+len(segments)+len(body))
	copy(page[:4], "OggS")
	page[26] = byte(len(segments))
	copy(page[27:], segments)
	copy(page[27+len(segments):], body)
	return page
}
