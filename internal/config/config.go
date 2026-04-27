package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultUpdateTimeout = 60 * time.Second
	defaultWorkDir       = ""
)

type Config struct {
	TelegramToken string
	SOCKS5Proxy   string
	WorkDir       string
	UpdateTimeout time.Duration
}

func Load() (Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	timeout := defaultUpdateTimeout
	if raw := os.Getenv("TELEGRAM_UPDATE_TIMEOUT"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("TELEGRAM_UPDATE_TIMEOUT must be a positive integer")
		}
		timeout = time.Duration(seconds) * time.Second
	}

	return Config{
		TelegramToken: token,
		SOCKS5Proxy:   os.Getenv("SOCKS5_PROXY"),
		WorkDir:       getenv("WORK_DIR", defaultWorkDir),
		UpdateTimeout: timeout,
	}, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
