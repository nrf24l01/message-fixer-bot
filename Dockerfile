# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/message-fixer-bot ./cmd/bot

FROM alpine:3.22

RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    tzdata \
    && addgroup -S app \
    && adduser -S -G app -h /app app

WORKDIR /app

COPY --from=builder /out/message-fixer-bot /usr/local/bin/message-fixer-bot

RUN mkdir -p /tmp/message-fixer-bot \
    && chown -R app:app /tmp/message-fixer-bot

USER app

ENV WORK_DIR=/tmp/message-fixer-bot

ENTRYPOINT ["message-fixer-bot"]
