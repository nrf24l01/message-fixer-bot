# Message Fixer Bot

Telegram bot that turns a replied voice message into a squeaky voice message when you send `/fix`, or a low voice message when you send `/serious_fix`.

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

## Docker Compose

The compose stack runs the bot plus an Xray SOCKS5 proxy. The bot connects to Telegram through `socks5://xray:1080`, and Xray builds its outbound config from `VLESS_URL` in `.env`.

```bash
cp .env.example .env
# edit TELEGRAM_BOT_TOKEN and VLESS_URL
docker compose up -d --build
```

Required `.env` values:

| Variable | Description |
| --- | --- |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token from BotFather. |
| `VLESS_URL` | Full `vless://` connection link for the Xray outbound. |

Optional `.env` values:

| Variable | Default | Description |
| --- | --- | --- |
| `TELEGRAM_UPDATE_TIMEOUT` | `60` | Bot long-poll timeout in seconds. |
| `XRAY_LOG_LEVEL` | `warning` | Xray log level. |

## Helm

The Helm chart deploys one pod with two containers: the bot and an Xray SOCKS5 sidecar. Store real credentials in Kubernetes secrets or pass them with a private values file.

```bash
helm lint helm/message-fixer-bot -f helm/message-fixer-bot/values.example.yaml
helm template demo helm/message-fixer-bot -f helm/message-fixer-bot/values.example.yaml
```

Install with an existing secret:

```bash
kubectl create secret generic message-fixer-bot-secret \
  --from-literal=TELEGRAM_BOT_TOKEN="<telegram-token>" \
  --from-literal=VLESS_URL="<vless-url>"

helm upgrade --install message-fixer-bot helm/message-fixer-bot \
  --set secrets.create=false \
  --set secrets.name=message-fixer-bot-secret
```

## Usage

1. Send a voice message to a chat with the bot.
2. Reply to that voice message with `/fix` for squeaky or `/serious_fix` for low.
3. The bot sends back the processed voice message.
