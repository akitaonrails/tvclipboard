# TV Clipboard

![Host Screen Example](static/img/tvclipboard.jpg)

A simple peer-to-peer clipboard sharing application written in Go. Share text between your desktop and mobile devices easily using a web interface and QR codes.

## Features

- 📱 **Device Agnostic**: Works on any device - desktop, tablet, or mobile
- 🔄 **Real-time**: Uses WebSockets for instant text transmission
- 📋 **Auto-Copy**: Automatically copies received text to clipboard
- 🔒 **Encryption**: AES-GCM encryption to prevent casual packet sniffing
- 👁️ **Privacy**: Password content obfuscated in received box (click to reveal)
- 🚀 **Simple**: No installation required on mobile devices
- 🔒 **P2P**: Direct connection between your devices
- 🎯 **Host/Client Roles**: First device becomes host, others are clients
- 👥 **Multi-Device**: Phone-to-phone, desktop-to-phone, any combination!
- ⏱️ **Session Management**: Time-limited QR codes prevent unauthorized access
- 🔄 **Auto-Refresh**: Host page regenerates QR code every session timeout
- 🛡️ **Rate Limiting**: Configurable message rate limits prevent abuse (default: 4 msg/sec)
- 📏 **Message Size Limits**: Configurable maximum message size prevents spam (default: 1KB)
- 🌐 **CORS Protection**: WebSocket origin validation using public URL config
- 🌍 **Internationalization**: Multi-language support (English and Brazilian Portuguese)

## Usage

### 1. Build the binary

```bash
go build -o tvclipboard
```

### 2. Run the server

```bash
./tvclipboard
```

The server will automatically detect your local IP address and start on port 3333. You'll see output like:

```
Server starting on port 3333
Local access: http://localhost:3333
Network access: http://192.168.1.100:3333
QR code will use: http://192.168.1.100:3333
Open in browser and scan QR code with your phone
```

### 3. Open in your browser

Navigate to any of the URLs shown in the console output (localhost works fine for desktop access)

### 4. Connect your mobile device

- The first device to connect becomes **Host** - it shows QR code and received text
- Scan the QR code with another device - it automatically opens as Client
- Additional devices can also open the URL with `?mode=client` to be Clients

### 5. Start sharing

- **From Host**: See QR code and received text box (text auto-copies to clipboard)
- **From Client**: Paste text and click "Send" - simple sending interface only
- Text automatically appears on the other device(s) and copies to clipboard

**How Roles Work:**

- **Host Mode**: First client to connect. Shows QR code and received content.
- **Client Mode**: Subsequent clients. Simplified interface for sending only.
- Works for phone-to-phone, desktop-to-phone, or any combination!
- **Only ONE host at a time**: The system allows only a single host connection. If you try to open host.html on another device while a host is already connected, it will be rejected with an error message.
- If the host disconnects, the next connected client becomes the new host.

## Tips

- All connected devices receive messages sent by any client
- Only the host sees the received messages box (other clients just send)
- Perfect for phone-to-phone clipboard sharing!
- Multiple clients can connect to the same host simultaneously
- **Encryption**: Messages are encrypted with AES-GCM (same default key on all devices)
  - For better security, edit the `sharedKey` variable in `static/index.html` line 312
- **Privacy**: Click on blurred received content to reveal it (protects passwords from prying eyes)
- **Paste Button Limitations**:
  - Browsers require HTTPS for automatic clipboard paste button
  - On `http://` (like local network), use long-press in textarea → "Paste"
  - Paste button works on `https://` or `localhost://`
- **Encryption and HTTPS**:
  - **Web Crypto API (AES-GCM encryption) is ONLY available in secure contexts**
  - Secure contexts: `https://`, `localhost` (any port), or `file://` URLs
  - When accessed via HTTP on LAN (e.g., `http://192.168.1.100:3333`), messages are sent **unencrypted**
  - This is a browser security restriction that cannot be bypassed
  - For encrypted messages in production, deploy with HTTPS (e.g., using a reverse proxy, ngrok, or self-signed certs)
  - For local development, unencrypted LAN access is typically acceptable (trusted network)
  - Check browser console: it will log "Web Crypto API available: false" in insecure contexts

## Architecture

The codebase is organized into focused, maintainable packages following Go best practices:

```
tvclipboard/
├── main.go           # Application entry point (wires packages together)
├── i18n/            # Internationalization (translations, language loading)
└── pkg/
    ├── token/         # Session management and encryption
    ├── hub/           # WebSocket hub and client management
    ├── qrcode/        # QR code generation
    ├── server/         # HTTP handlers and routing
    └── config/        # Configuration and environment variables
└── static/
    ├── js/            # Frontend JavaScript (i18n, common, host, client)
    ├── host.html      # Host page UI
    ├── client.html    # Client page UI
    └── img/          # Static images
```

### Package Responsibilities

- **pkg/token**: Session token generation, AES-GCM encryption, validation, cleanup
- **pkg/hub**: WebSocket connection management, message broadcasting, role assignment
- **pkg/qrcode**: QR code generation, HTML injection for session timeout
- **pkg/server**: HTTP route handlers, WebSocket upgrades, static file serving, i18n injection
- **pkg/config**: CLI argument parsing, environment variable parsing, IP detection, startup logging, language selection
- **i18n**: Translation loading (YAML), server-side i18n management, JSON injection

### Frontend Architecture

- **static/js/i18n.js**: Translation lookup, placeholder substitution, DOM translation application
- **static/js/common.js**: Shared utilities (WebSocket URLs, formatting, encryption)
- **static/js/host.js**: Host-specific logic (QR code, message reception, reveal toggle)
- **static/js/client.js**: Client-specific logic (message sending, session management)

## Testing

TV Clipboard includes a comprehensive test suite covering both Go backend and JavaScript frontend:

### Running Tests

```bash
# Run all Go tests
go test ./... -v

# Run Go tests with coverage
go test ./... -cover

# Run specific Go test
go test ./pkg/token -v -run TestTokenGeneration

# Run JavaScript tests
npm test

# Run all tests (Go and JavaScript)
go test ./... && npm test
```

### Go Test Coverage

The Go test suite includes 62 tests covering:

**pkg/config (6 tests)**

- Default configuration loading
- Environment variable configuration
- CLI argument configuration
- CLI flags override environment variables
- Invalid/zero/negative timeout handling

**pkg/token (17 tests)**

- Token generation with valid format
- Token encryption/decryption with AES-GCM
- Token validation (valid, invalid, expired, not found)
- Token cleanup of expired sessions
- Private key generation from hex string or random
- Token timeout configuration
- Multiple concurrent tokens
- Token JSON encoding/decoding

**pkg/hub (12 tests)**

- Message broadcasting to all clients except sender
- Concurrent message handling
- Client connection and reconnection
- Long message handling (10KB)
- Message type validation (text, role)
- Message size limits (up to 100KB)
- Empty messages
- Messages with quotes and special characters
- Multiline messages
- Encryption compatibility with various content types

**pkg/qrcode (5 tests)**

- QR code endpoint returns valid PNG format
- QR code URL contains proper token parameter
- QR code generator configuration
- Session timeout HTML injection
- HTML replacement utilities

**pkg/server (9 tests)**

- Host connection without token succeeds
- Host connection with token is rejected
- Client connection without token is rejected when host exists
- Client connection with invalid/expired token is rejected
- QR code endpoint generation
- Client URL handling
- Cache busting version injection
- Version format validation (YYYYMMDDHHMMSS)

### JavaScript Test Coverage

The JavaScript test suite includes 57 headless tests covering:

**i18n (18 tests)**

- Translation lookup with fallback
- Placeholder substitution (single, multiple, repeated)
- Section-based and section-less keys
- Missing keys/sections handling
- Null translations and null/undefined values
- Numbers and special characters in params
- Empty strings and params objects

**common (4 tests)**

- WebSocket URL generation (http/https, custom ports)
- Public URL retrieval
- Time formatting (seconds to MM:SS)
- Edge cases (hour boundaries, single digits)

**websocket (35 tests)**

- Message structure and serialization
- Role assignment (host/client)
- WebSocket states (CONNECTING, OPEN, CLOSING, CLOSED)
- Error codes (1000, 1006, 4000+)
- Connection state transitions
- Session expiration and connection failure tracking
- Content validation (empty, whitespace-only)
- Content obfuscation and reveal toggle
- Timer logic (decrement, warning thresholds)
- JSON handling (Unicode, special chars, empty, very long)
- Complex message workflow simulation
- URL with token parameter
- Reconnection delay
- Message type validation

### Test Approach

**Go Tests**: Unit tests covering all business logic, including concurrency, error handling, and edge cases.

**JavaScript Tests**: Headless tests using Node.js built-in test runner. Tests public functions and workflow logic without requiring browser APIs or DOM manipulation. This keeps tests fast and simple while covering critical functionality.

### Code Quality Tools

#### Linting

```bash
# Run Go linter (requires golangci-lint)
golangci-lint run

# Run JavaScript linter
npm run lint

# Run both linters
golangci-lint run && npm run lint
```

**Go Linting**: golangci-lint is configured in `.golangci.yml` to check for code quality issues, potential bugs, and style inconsistencies.

**JavaScript Linting**: ESLint is configured in `eslint.config.js` to catch bugs and logic errors without enforcing style preferences. Rules focus on:
- Possible errors (no-constant-condition, no-dupe-keys, no-unsafe-finally, etc.)
- Best practices (eqeqeq, no-eval, no-with, radix, etc.)
- Variables (no-undef, no-unused-vars, no-redeclare, etc.)

No formatting or whitespace rules are enforced - only actual bugs and logic errors.

#### Lines of Code

```bash
# Count Go code
./loc.sh

# Count JavaScript code
./loc-js.sh

# Show both
./loc.sh && echo "---" && ./loc-js.sh
```

LOC counters show:
- Total lines of code
- Production vs test code breakdown
- Test ratio (test lines : production lines)
- Largest files by line count

**Current stats**:
- Go: 3,493 lines (1,431 production, 2,062 tests, 1.44:1 ratio)
- JavaScript: 1,360 lines (787 production, 573 tests, 0.73:1 ratio)
- Total: 4,853 lines (2,218 production, 2,635 tests, 1.19:1 ratio)

### Test Coverage

The test suite includes 62 tests covering:

**pkg/config (6 tests)**

- Default configuration loading
- Environment variable configuration
- CLI argument configuration
- CLI flags override environment variables
- Invalid/zero/negative timeout handling

**pkg/token (17 tests)**

- Token generation with valid format
- Token encryption/decryption with AES-GCM
- Token validation (valid, invalid, expired, not found)
- Token cleanup of expired sessions
- Private key generation from hex string or random
- Token timeout configuration
- Multiple concurrent tokens
- Token JSON encoding/decoding

**pkg/hub (12 tests)**

- Message broadcasting to all clients except sender
- Concurrent message handling
- Client connection and reconnection
- Long message handling (10KB)
- Message type validation (text, role)
- Message size limits (up to 100KB)
- Empty messages
- Messages with quotes and special characters
- Multiline messages
- Encryption compatibility with various content types

**pkg/qrcode (5 tests)**

- QR code endpoint returns valid PNG format
- QR code URL contains proper token parameter
- QR code generator configuration
- Session timeout HTML injection
- HTML replacement utilities

**pkg/server (9 tests)**

- Host connection without token succeeds
- Host connection with token is rejected
- Client connection without token is rejected when host exists
- Client connection with invalid/expired token is rejected
- QR code endpoint generation
- Client URL handling
- Cache busting version injection
- Version format validation (YYYYMMDDHHMMSS)

### Test Coverage

Current test coverage focuses on:

**Go Backend**:
- Session token generation and validation
- WebSocket connection lifecycle
- Message serialization/deserialization
- Token-based authentication
- QR code generation
- HTTP route handlers and static file serving
- I18n translation loading and injection

**JavaScript Frontend**:
- Translation lookup and placeholder substitution
- WebSocket URL generation
- Time formatting
- Message structure validation
- Workflow state management (connection, session, expiration)
- JSON serialization/deserialization
- Content validation and obfuscation

**Not Covered** (by design):
- Client-side Web Crypto API (browser-vendor tested)
- DOM manipulation (requires JSDOM, adds complexity)
- Clipboard API (browser-only APIs)
- WebSocket network connections (requires browser)

Note: Client-side encryption (using Web Crypto API in JavaScript) is tested through integration tests that verify complete message flow between clients and host. Browser APIs are tested by browser vendors, so we focus on business logic and workflow testing.

The main.go file is minimal (~47 lines) and only wires the packages together. All business logic is encapsulated in focused packages for maintainability and testability.

## Requirements

- Go 1.16 or higher (for building)
- Node.js 18+ (for running JavaScript tests)
- Modern web browser with JavaScript support

## Dependencies

### Go Dependencies

- github.com/gorilla/websocket
- github.com/google/uuid
- github.com/skip2/go-qrcode
- gopkg.in/yaml.v3 (for i18n translation files)

### JavaScript Dependencies (dev only)

- eslint - JavaScript linter for catching bugs and logic errors
- globals - Browser globals for ESLint configuration

JavaScript dependencies are only used for testing and linting during development. The production build is a single Go binary with embedded static files.

## Security

### Session Management

TV Clipboard includes session-based security to prevent unauthorized access:

- **Time-Limited QR Codes**: Each QR code contains an encrypted token that expires after a configurable timeout (default: 10 minutes)
- **Token Validation**: WebSocket connections must provide a valid, non-expired token
- **Auto-Refresh**: Host page automatically refreshes and generates a new QR code before session expires
- **Client Expiration**: Clients show a countdown timer and disable sending when session expires

### Environment Variables

You can configure session security using environment variables:

#### `TVCLIPBOARD_PRIVATE_KEY`

- 32-byte hexadecimal string used to encrypt session tokens
- If not set, a random key is generated on each server restart
- Example: `TVCLIPBOARD_PRIVATE_KEY="a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"`

#### `TVCLIPBOARD_SESSION_TIMEOUT`

- Session timeout in minutes (integer)
- Default: 10 minutes
- Example: `TVCLIPBOARD_SESSION_TIMEOUT=15`

#### `TVCLIPBOARD_PUBLIC_URL`

- Public base URL for QR codes (full URL or just scheme://host)
- If not set, automatically detects local IP address
- Useful for deployments with public domain names
- Example: `TVCLIPBOARD_PUBLIC_URL=https://example.com`
- Example: `TVCLIPBOARD_PUBLIC_URL=https://example.com:3333`

#### `TVCLIPBOARD_MAX_MESSAGE_SIZE`

- Maximum message size in kilobytes (integer)
- Default: 1 KB (approximately 1000 characters)
- Messages exceeding this limit are rejected and logged
- Example: `TVCLIPBOARD_MAX_MESSAGE_SIZE=1`

#### `TVCLIPBOARD_RATE_LIMIT`

- Maximum number of messages per second per client (integer)
- Default: 4 messages per second
- Clients exceeding this limit receive error messages
- Example: `TVCLIPBOARD_RATE_LIMIT=4`

### Usage Examples

**Option 1: Environment Variables**

```bash
# Set custom private key and 15-minute timeout
export TVCLIPBOARD_PRIVATE_KEY="your-32-byte-hex-key-here"
export TVCLIPBOARD_SESSION_TIMEOUT=15
./tvclipboard
```

**Option 2: CLI Arguments (simpler for direct usage)**

```bash
# Show help
./tvclipboard --help

# Run with custom port and session timeout
./tvclipboard --port 9999 --expires 5

# Run with custom private key
./tvclipboard --key "deadbeef1234567890abcdef1234567890"

# Run with public domain for QR codes
./tvclipboard --base-url "https://example.com"

# Set rate limiting and message size
./tvclipboard --rate-limit 4 --max-message-size 1

# Combine all options
./tvclipboard --port 8080 --base-url "https://example.com" --expires 15 --key "your-key-here" --rate-limit 4 --max-message-size 1
```

**Configuration Priority:** CLI flags override environment variables, which override defaults.

### Token Encryption

Session tokens are encrypted using AES-GCM with your private key:

- Each token contains a random UUID and timestamp
- Tokens are stored server-side and validated on connection
- Expired tokens are automatically cleaned up every minute
- Invalid or expired connections are rejected with HTTP 401

### Security Notes

- Tokens are included in QR code URLs, so anyone who scans QR code can connect
- Session timeout limits how long a QR code remains valid
- Private key rotation requires server restart (new QR code generation)
- For local network use, default settings provide reasonable security

## License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

### What AGPL Means

This is a copyleft license that:

- ✅ Allows you to freely use, modify, and distribute the software
- ✅ Requires you to share your modifications under the same license
- ✅ Requires network deployments to provide source code to users
- ❌ Does NOT allow you to close the source or make it proprietary

If you modify this program and run it as a network service, your users must have access to the source code of your modified version.

### Quick Summary

- **Commercial use**: Allowed
- **Modifying**: Allowed (must share modifications)
- **Distributing**: Allowed (must use same license)
- **Sublicensing**: Not allowed
- **Liability**: Disclaimed (use at your own risk)

For full license text, see [LICENSE](LICENSE) file.

## Copyright

Copyright (C) 2026 Fabio Akita

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation.
