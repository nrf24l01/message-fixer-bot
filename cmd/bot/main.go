package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nrf24l01/message-fixer-bot/internal/app"
	"github.com/nrf24l01/message-fixer-bot/internal/config"
	"github.com/nrf24l01/message-fixer-bot/internal/service"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "run container healthcheck")
	flag.Parse()

	if *healthcheck {
		if err := runHealthcheck(); err != nil {
			fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

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

func runHealthcheck() error {
	processor := service.NewVoiceProcessor()
	if err := processor.CheckFFmpeg(); err != nil {
		return err
	}

	proxyURL := os.Getenv("SOCKS5_PROXY")
	if proxyURL == "" {
		return nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("parse SOCKS5_PROXY: %w", err)
	}
	if parsed.Host == "" {
		return fmt.Errorf("SOCKS5_PROXY must include host")
	}

	conn, err := net.DialTimeout("tcp", parsed.Host, 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect to SOCKS5 proxy: %w", err)
	}
	return conn.Close()
}
