# TV Clipboard

A simple peer-to-peer clipboard sharing application written in Go. Share text between your desktop and mobile devices easily using a web interface and QR codes.

## Features

- 📱 **Device Agnostic**: Works on any device - desktop, tablet, or mobile
- 🔄 **Real-time**: Uses WebSockets for instant text transmission
- 📋 **Auto-Copy**: Automatically copies received text to clipboard
- 🚀 **Simple**: No installation required on mobile devices
- 🔒 **P2P**: Direct connection between your devices
- 🎯 **Host/Client Roles**: First device becomes host, others are clients
- 👥 **Multi-Device**: Phone-to-phone, desktop-to-phone, any combination!

## Usage

### 1. Build the binary
```bash
go build -o tvclipboard
```

### 2. Run the server
```bash
./tvclipboard
```

The server will automatically detect your local IP address and start on port 8080. You'll see output like:
```
Server starting on port 8080
Local access: http://localhost:8080
Network access: http://192.168.1.100:8080
QR code will use: http://192.168.1.100:8080
Open in browser and scan QR code with your phone
```

### 3. Open in your browser
Navigate to any of the URLs shown in the console output (localhost works fine for desktop access)

### 4. Connect your mobile device
- The first device to connect becomes **Host** - it shows the QR code and received text
- Scan the QR code with another device or open the network URL
- Additional devices connect as **Clients** - they have a simple interface for sending only

### 5. Start sharing!
- **From Host**: See the QR code and received text box (text auto-copies to clipboard) - no input area
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

## Requirements

- Go 1.16 or higher (for building)
- Modern web browser with JavaScript support

## Dependencies

- github.com/gorilla/websocket
- github.com/google/uuid
- github.com/skip2/go-qrcode
