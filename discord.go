package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type discordPlayer struct {
	session *discordgo.Session
	voice   *discordgo.VoiceConnection
}

func connectDiscord(ctx context.Context, cfg config) (*discordPlayer, error) {
	session, err := discordgo.New("Bot " + cfg.discordToken)
	if err != nil {
		return nil, fmt.Errorf("create Discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates

	ready := make(chan struct{})
	var readyOnce sync.Once
	session.AddHandler(func(*discordgo.Session, *discordgo.Ready) {
		readyOnce.Do(func() { close(ready) })
	})
	if err := session.Open(); err != nil {
		return nil, errors.Join(fmt.Errorf("open Discord session: %w", err), session.Close())
	}

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case <-ready:
	case <-ctx.Done():
		return nil, errors.Join(ctx.Err(), session.Close())
	case <-timer.C:
		return nil, errors.Join(fmt.Errorf("timed out waiting for Discord READY event"), session.Close())
	}

	joinCtx, cancelJoin := context.WithTimeout(ctx, 30*time.Second)
	voice, err := session.ChannelVoiceJoin(joinCtx, cfg.discordGuildID, cfg.voiceChannelID, false, true)
	cancelJoin()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("join configured Discord voice channel: %w", err), session.Close())
	}

	daveCtx, cancelDAVE := context.WithTimeout(ctx, 30*time.Second)
	defer cancelDAVE()
	if err := voice.WaitForDAVEReady(daveCtx); err != nil {
		disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDisconnect()
		return nil, errors.Join(
			fmt.Errorf("wait for Discord DAVE encryption: %w", err),
			voice.Disconnect(disconnectCtx),
			session.Close(),
		)
	}
	return &discordPlayer{session: session, voice: voice}, nil
}

func (p *discordPlayer) play(ctx context.Context, path, ffmpegPath string) error {
	if err := p.voice.Speaking(true); err != nil {
		return fmt.Errorf("enable Discord speaking: %w", err)
	}
	streamErr := streamOpus(ctx, path, ffmpegPath, func(packet []byte) error {
		select {
		case p.voice.OpusSend <- packet:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	if err := p.voice.Speaking(false); err != nil {
		return errors.Join(streamErr, fmt.Errorf("disable Discord speaking: %w", err))
	}
	return streamErr
}

func (p *discordPlayer) Close() error {
	var disconnectErr error
	if p.voice != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		disconnectErr = p.voice.Disconnect(ctx)
		cancel()
	}
	closeErr := p.session.Close()
	if disconnectErr != nil {
		disconnectErr = fmt.Errorf("disconnect Discord voice: %w", disconnectErr)
	}
	return errors.Join(disconnectErr, closeErr)
}
