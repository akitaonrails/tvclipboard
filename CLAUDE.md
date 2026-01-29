# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TV Clipboard is a peer-to-peer clipboard sharing application written in Go with a vanilla JavaScript frontend. It enables real-time text sharing between devices using WebSockets, with QR code-based session management and AES-GCM encryption.

## Build & Run Commands

```bash
# Build
go build -o tvclipboard

# Run (default port 3333)
./tvclipboard

# Run with options
./tvclipboard --port 8080 --expires 15 --base-url "https://example.com"
```

## Testing

```bash
# All Go tests
go test ./... -v

# Go tests with coverage
go test ./... -cover

# Single Go test
go test ./pkg/token -v -run TestTokenGeneration

# JavaScript tests (Node.js 18+)
npm test

# Full test suite
go test ./... && npm test
```

## Linting

```bash
# Go linting
golangci-lint run

# JavaScript linting
npm run lint
```

## Docker

```bash
docker build -t tvclipboard:latest .
docker-compose up -d
```

## Architecture

### Entry Point
`main.go` - Loads config, initializes i18n, wires together Hub/TokenManager/QRGenerator/Server, handles graceful shutdown.

### Core Packages (pkg/)

- **config/** - CLI flags, env vars, startup configuration. Priority: CLI > env vars > defaults.
- **token/** - Session token generation with AES-GCM encryption, validation, auto-cleanup of expired tokens.
- **hub/** - WebSocket connection manager, message broadcasting, rate limiting. Single host model (only one host connected at a time).
- **qrcode/** - QR code PNG generation as base64 data URIs.
- **server/** - HTTP handlers, WebSocket upgrades, static file serving, CORS validation, i18n injection into HTML templates.

### Internationalization
- **i18n/** - Translation loading from YAML files.
- **langs/** - Translation files (en.yml, pt_br.yml).
- Singleton pattern: `i18n.GetInstance()`.

### Frontend (static/)
- **i18n.js** - Translation function `t(key, params)` with fallback handling.
- **common.js** - AES-GCM encryption via Web Crypto API, WebSocket URL construction.
- **host.js** - QR display, message reception, content visibility toggle.
- **client.js** - Message sending, session countdown, reconnection logic.

## Configuration

Environment variables (all have corresponding CLI flags):
- `PORT` - Server port (default: 3333)
- `TVCLIPBOARD_SESSION_TIMEOUT` - Session timeout in minutes (default: 10)
- `TVCLIPBOARD_PRIVATE_KEY` - 32-byte hex key for token encryption
- `TVCLIPBOARD_PUBLIC_URL` - Public base URL for QR codes
- `TVCLIPBOARD_MAX_MESSAGE_SIZE` - Max message size in KB (default: 1)
- `TVCLIPBOARD_RATE_LIMIT` - Messages per second per client (default: 4)
- `TVCLIPBOARD_LANGUAGE` - Language code (default: en)

## Key Design Decisions

- **Single host model**: Only one host can be connected per session.
- **In-memory state**: No database; tokens and clients stored in memory with mutex protection.
- **Web Crypto limitation**: AES-GCM encryption requires HTTPS or localhost (browser security restriction). Messages are unencrypted on http://192.168.x.x.
- **Token expiration**: Cleaned up every minute via background goroutine.
