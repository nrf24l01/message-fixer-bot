package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/net/proxy"

	"github.com/nrf24l01/message-fixer-bot/internal/config"
	"github.com/nrf24l01/message-fixer-bot/internal/service"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	processor *service.VoiceProcessor
	timeout   time.Duration
	workDir   string
}

func NewBot(cfg config.Config) (*Bot, error) {
	log.Printf("creating telegram client")
	if cfg.SOCKS5Proxy != "" {
		log.Printf("using SOCKS5 proxy: %s", cfg.SOCKS5Proxy)
	}

	client, err := httpClient(cfg.SOCKS5Proxy)
	if err != nil {
		return nil, err
	}

	api, err := tgbotapi.NewBotAPIWithClient(cfg.TelegramToken, tgbotapi.APIEndpoint, client)
	if err != nil {
		return nil, fmt.Errorf("create telegram client: %w", err)
	}

	workDir := cfg.WorkDir
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "message-fixer-bot-*")
		if err != nil {
			return nil, fmt.Errorf("create work dir: %w", err)
		}
	} else if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}

	processor := service.NewVoiceProcessor()
	log.Printf("checking ffmpeg availability")
	if err := processor.CheckFFmpeg(); err != nil {
		return nil, err
	}
	log.Printf("ffmpeg is available")

	return &Bot{
		api:       api,
		processor: processor,
		timeout:   cfg.UpdateTimeout,
		workDir:   workDir,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	log.Printf("authorized as @%s", b.api.Self.UserName)
	log.Printf("starting polling with timeout %s", b.timeout)

	updates := b.api.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  0,
		Timeout: int(b.timeout.Seconds()),
	})

	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping polling")
			b.api.StopReceivingUpdates()
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				log.Printf("updates channel closed")
				return nil
			}
			if update.Message == nil {
				continue
			}
			logMessage(update.Message)
			if err := b.handleMessage(ctx, update.Message); err != nil {
				log.Printf("handle message %d: %v", update.Message.MessageID, err)
			}
		}
	}
}

func logMessage(msg *tgbotapi.Message) {
	from := "unknown"
	if msg.From != nil {
		from = fmt.Sprintf("%s(id=%d username=%s)", msg.From.FirstName, msg.From.ID, msg.From.UserName)
	}

	log.Printf(
		"message received chat_id=%d chat_type=%s message_id=%d from=%s kind=%s command=%q text=%q reply_to=%d",
		msg.Chat.ID,
		msg.Chat.Type,
		msg.MessageID,
		from,
		messageKind(msg),
		msg.Command(),
		msg.Text,
		replyToMessageID(msg),
	)
}

func messageKind(msg *tgbotapi.Message) string {
	switch {
	case msg.Voice != nil:
		return fmt.Sprintf("voice(duration=%d file_id=%s)", msg.Voice.Duration, msg.Voice.FileID)
	case msg.Audio != nil:
		return fmt.Sprintf("audio(duration=%d file_id=%s)", msg.Audio.Duration, msg.Audio.FileID)
	case msg.Video != nil:
		return fmt.Sprintf("video(duration=%d file_id=%s)", msg.Video.Duration, msg.Video.FileID)
	case msg.VideoNote != nil:
		return fmt.Sprintf("video_note(duration=%d file_id=%s)", msg.VideoNote.Duration, msg.VideoNote.FileID)
	case msg.Document != nil:
		return fmt.Sprintf("document(file_name=%s file_id=%s)", msg.Document.FileName, msg.Document.FileID)
	case len(msg.Photo) > 0:
		photo := msg.Photo[len(msg.Photo)-1]
		return fmt.Sprintf("photo(file_id=%s)", photo.FileID)
	case msg.Sticker != nil:
		return fmt.Sprintf("sticker(emoji=%s file_id=%s)", msg.Sticker.Emoji, msg.Sticker.FileID)
	case msg.Location != nil:
		return "location"
	case msg.Contact != nil:
		return "contact"
	case msg.Text != "":
		return "text"
	default:
		return "unknown"
	}
}

func replyToMessageID(msg *tgbotapi.Message) int {
	if msg.ReplyToMessage == nil {
		return 0
	}
	return msg.ReplyToMessage.MessageID
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) error {
	command := msg.Command()
	if command != "fix" && command != "serious_fix" {
		return nil
	}
	log.Printf("received /%s in chat %d message %d", command, msg.Chat.ID, msg.MessageID)

	voiceMsg := msg
	if msg.ReplyToMessage != nil {
		voiceMsg = msg.ReplyToMessage
	}
	if voiceMsg.Voice == nil {
		log.Printf("/%s message %d has no voice target", command, msg.MessageID)
		prompt := fmt.Sprintf("Reply to a voice message with /%s.", command)
		_, err := b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, prompt))
		return err
	}

	log.Printf("processing voice file_id=%s duration=%d", voiceMsg.Voice.FileID, voiceMsg.Voice.Duration)
	if _, err := b.api.Request(tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatUploadVoice)); err != nil {
		return fmt.Errorf("send chat action: %w", err)
	}

	inputPath, err := b.downloadVoice(voiceMsg.Voice.FileID)
	if err != nil {
		return err
	}
	defer os.Remove(inputPath)
	log.Printf("downloaded voice to %s", inputPath)

	outputPath := filepath.Join(b.workDir, fmt.Sprintf("fixed-%d.ogg", time.Now().UnixNano()))
	defer os.Remove(outputPath)

	log.Printf("running ffmpeg for message %d", msg.MessageID)
	var processErr error
	if command == "serious_fix" {
		processErr = b.processor.MakeSerious(ctx, inputPath, outputPath)
	} else {
		processErr = b.processor.MakeSqueaky(ctx, inputPath, outputPath)
	}
	if processErr != nil {
		log.Printf("ffmpeg failed for message %d: %v", msg.MessageID, processErr)
		_, sendErr := b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Could not process this voice message. Make sure ffmpeg is installed."))
		if sendErr != nil {
			return errors.Join(processErr, sendErr)
		}
		return processErr
	}
	log.Printf("created fixed voice at %s", outputPath)

	voice := tgbotapi.NewVoice(msg.Chat.ID, tgbotapi.FilePath(outputPath))
	voice.ReplyToMessageID = msg.MessageID
	if _, err := b.api.Send(voice); err != nil {
		return err
	}
	log.Printf("sent fixed voice for message %d", msg.MessageID)
	return nil
}

func (b *Bot) downloadVoice(fileID string) (string, error) {
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("get telegram file: %w", err)
	}
	log.Printf("telegram file path: %s size=%d", file.FilePath, file.FileSize)

	fileURL := file.Link(b.api.Token)
	req, err := http.NewRequest(http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}

	resp, err := b.api.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download voice: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download voice: unexpected status %s", resp.Status)
	}

	path := filepath.Join(b.workDir, fmt.Sprintf("voice-%d.oga", time.Now().UnixNano()))
	out, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create voice file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("save voice file: %w", err)
	}

	return path, nil
}

func httpClient(proxyURL string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL == "" {
		return &http.Client{Transport: transport}, nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse SOCKS5_PROXY: %w", err)
	}
	if parsed.Scheme != "socks5" && parsed.Scheme != "socks5h" {
		return nil, fmt.Errorf("SOCKS5_PROXY must use socks5 or socks5h scheme")
	}

	auth := &proxy.Auth{}
	if parsed.User != nil {
		auth.User = parsed.User.Username()
		auth.Password, _ = parsed.User.Password()
	} else {
		auth = nil
	}

	dialer, err := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("create socks5 dialer: %w", err)
	}

	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("socks5 dialer does not support context")
	}

	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return contextDialer.DialContext(ctx, network, address)
	}

	return &http.Client{Transport: transport}, nil
}
