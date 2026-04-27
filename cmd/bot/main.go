package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nrf24l01/message-fixer-bot/internal/app"
	"github.com/nrf24l01/message-fixer-bot/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot, err := app.NewBot(cfg)
	if err != nil {
		log.Fatalf("create bot: %v", err)
	}

	if err := bot.Run(ctx); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "bot stopped: %v\n", err)
		os.Exit(1)
	}
}
