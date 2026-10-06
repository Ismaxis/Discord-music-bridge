package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func streamOpus(ctx context.Context, inputPath, ffmpegPath string, send func([]byte) error) error {
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-nostdin", "-v", "error", "-i", inputPath,
		"-vn", "-ac", "2", "-ar", "48000", "-c:a", "libopus",
		"-b:a", "128k", "-frame_duration", "20", "-f", "ogg", "pipe:1",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open ffmpeg output: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg (is it installed?): %w", err)
	}

	parseErr := readOpusPackets(stdout, send)
	var killErr error
	if parseErr != nil {
		killErr = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if parseErr != nil {
		return errors.Join(fmt.Errorf("read ffmpeg audio stream: %w", parseErr), killErr)
	}
	if waitErr != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", waitErr, stderr.String())
	}
	return nil
}

func readOpusPackets(r io.Reader, send func([]byte) error) error {
	var packet []byte
	headerPackets := 0
	audioPackets := 0
	for {
		var pageHeader [27]byte
		_, err := io.ReadFull(r, pageHeader[:])
		if err == io.EOF {
			if headerPackets < 2 {
				return fmt.Errorf("audio stream ended before Opus headers")
			}
			if len(packet) != 0 {
				return fmt.Errorf("audio stream ended in the middle of an Ogg packet")
			}
			if audioPackets == 0 {
				return fmt.Errorf("audio stream contains no Opus frames")
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read Ogg page header: %w", err)
		}
		if string(pageHeader[:4]) != "OggS" || pageHeader[4] != 0 {
			return fmt.Errorf("invalid Ogg page")
		}

		segments := make([]byte, int(pageHeader[26]))
		if _, err := io.ReadFull(r, segments); err != nil {
			return fmt.Errorf("read Ogg segment table: %w", err)
		}
		bodySize := 0
		for _, size := range segments {
			bodySize += int(size)
		}
		body := make([]byte, bodySize)
		if _, err := io.ReadFull(r, body); err != nil {
			return fmt.Errorf("read Ogg page body: %w", err)
		}

		offset := 0
		for _, size := range segments {
			packet = append(packet, body[offset:offset+int(size)]...)
			offset += int(size)
			if size < 255 {
				if headerPackets < 2 {
					expected := "OpusHead"
					if headerPackets == 1 {
						expected = "OpusTags"
					}
					if len(packet) < len(expected) || string(packet[:len(expected)]) != expected {
						return fmt.Errorf("Ogg stream is missing the %s packet", expected)
					}
					headerPackets++
				} else {
					frame := append([]byte(nil), packet...)
					if err := send(frame); err != nil {
						return err
					}
					audioPackets++
				}
				packet = packet[:0]
			}
		}
	}
}
