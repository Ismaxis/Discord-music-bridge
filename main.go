package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	discord, err := connectDiscord(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := discord.Close(); err != nil {
			log.Printf("close Discord session: %v", err)
		}
	}()

	bot := newTelegramBot(cfg, discord)
	if err := bot.run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
