# Message Fixer Bot

Telegram bot that turns a replied voice message into a squeaky voice message when you send `/fix`.

## Requirements

- Go 1.22+
- `ffmpeg` available in `PATH`
- Telegram bot token from BotFather

## Configuration

Environment variables:

| Variable | Required | Description |
| --- | --- | --- |
| `TELEGRAM_BOT_TOKEN` | yes | Telegram bot token. |
| `SOCKS5_PROXY` | no | SOCKS5 proxy URL, for example `socks5://127.0.0.1:1080` or `socks5://user:pass@127.0.0.1:1080`. |
| `TELEGRAM_UPDATE_TIMEOUT` | no | Long-poll timeout in seconds. Defaults to `60`. |
| `WORK_DIR` | no | Directory for temporary voice files. Defaults to a generated temp directory. |

## Run

```bash
export TELEGRAM_BOT_TOKEN="123456:token"
export SOCKS5_PROXY="socks5://127.0.0.1:1080"
go run ./cmd/bot
```

## Usage

1. Send a voice message to a chat with the bot.
2. Reply to that voice message with `/fix`.
3. The bot sends back a squeaky version of the voice message.
