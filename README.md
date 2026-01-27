# TV Clipboard

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

### 5. Start sharing!
- **From Host**: See QR code and received text box (text auto-copies to clipboard)
- **From Client**: Paste text and click "Send" - simple sending interface only
- Text automatically appears on the other device(s) and copies to clipboard

**How Roles Work:**
- **Host Mode**: First client to connect. Shows QR code and received content.
- **Client Mode**: Subsequent clients. Simplified interface for sending only.
- Works for phone-to-phone, desktop-to-phone, or any combination!
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

## Architecture

The codebase is organized into focused, maintainable packages following Go best practices:

```
tvclipboard/
├── main.go           # Application entry point (wires packages together)
└── pkg/
    ├── token/         # Session management and encryption
    ├── hub/           # WebSocket hub and client management
    ├── qrcode/        # QR code generation
    ├── server/         # HTTP handlers and routing
    └── config/        # Configuration and environment variables
```

### Package Responsibilities

- **pkg/token**: Session token generation, AES-GCM encryption, validation, cleanup
- **pkg/hub**: WebSocket connection management, message broadcasting, role assignment
- **pkg/qrcode**: QR code generation, HTML injection for session timeout
- **pkg/server**: HTTP route handlers, WebSocket upgrades, static file serving
- **pkg/config**: CLI argument parsing, environment variable parsing, IP detection, startup logging

## Testing

TV Clipboard includes a comprehensive test suite covering all major functionality:

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Run specific test
go test ./pkg/token -v -run TestTokenGeneration
```

### Test Coverage

The test suite includes 54 tests covering:

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
- Session token generation and validation
- WebSocket connection lifecycle
- Message serialization/deserialization
- Token-based authentication
- QR code generation

Note: Client-side encryption (using Web Crypto API in JavaScript) is tested through integration tests that verify complete message flow between clients and host.

The main.go file is minimal (~47 lines) and only wires the packages together. All business logic is encapsulated in focused packages for maintainability and testability.

## Requirements

- Go 1.16 or higher (for building)
- Modern web browser with JavaScript support

## Dependencies

- github.com/gorilla/websocket
- github.com/google/uuid
- github.com/skip2/go-qrcode

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

# Combine all options
./tvclipboard --port 8080 --expires 15 --key "your-key-here"
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
