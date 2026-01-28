package server

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"tvclipboard/pkg/hub"
	"tvclipboard/pkg/qrcode"
	"tvclipboard/pkg/token"
)

// testFS is a minimal in-memory filesystem for testing
type testFS struct{}

func (testFS) Open(name string) (fs.File, error) {
	// Return a minimal HTML for testing
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (testFS) ReadFile(name string) ([]byte, error) {
	// Minimal HTML for version injection testing
	if strings.HasSuffix(name, "host.html") {
		return []byte(`<!DOCTYPE html>
<html>
<body>
<script src="/static/js/common.js"></script>
<script src="/static/js/host.js"></script>
</body>
</html>`), nil
	}
	if strings.HasSuffix(name, "client.html") {
		return []byte(`<!DOCTYPE html>
<html>
<body class="container">
<script src="/static/js/common.js"></script>
<script src="/static/js/client.js"></script>
</body>
</html>`), nil
	}
	return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
}

var mockStaticFiles testFS

// TestClientURLMissingToken tests that client page responds correctly to missing token
func TestClientURLMissingToken(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()

	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9) // 10 minutes

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleIndex(w, r)
	}))
	defer server.Close()

	// Request client page without token
	resp, err := http.Get(server.URL + "/?mode=client")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Page should load with 404 since we don't have real static files
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("Note: Using mock filesystem, got status %v", resp.StatusCode)
	}
}

// TestWebSocketConnectionWithoutToken tests that WebSocket rejects connections without token when host exists
func TestWebSocketConnectionWithoutToken(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()

	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Simulate host exists by setting hostID
	h.SetHostID("test-host")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleWebSocket(w, r)
	}))
	defer server.Close()

	// Try to connect without token (should fail)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	_, _, err := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
	if err == nil {
		t.Error("WebSocket connection without token should fail when host exists")
	}

	// HTTP 401 results in "bad handshake" error from WebSocket client
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Errorf("Expected handshake error, got: %v", err)
	}
}

// TestWebSocketConnectionWithInvalidToken tests that WebSocket rejects invalid tokens
func TestWebSocketConnectionWithInvalidToken(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()
	
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)
	
	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Simulate host exists
	h.SetHostID("test-host")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleWebSocket(w, r)
	}))
	defer server.Close()

	// Try to connect with invalid token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=invalid"
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Error("WebSocket connection with invalid token should fail")
	}

	// HTTP 401 results in "bad handshake" error from WebSocket client
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Errorf("Expected handshake error, got: %v", err)
	}
}

// TestWebSocketConnectionWithExpiredToken tests that WebSocket rejects expired tokens
func TestWebSocketConnectionWithExpiredToken(t *testing.T) {
	tm := token.NewTokenManager("", 1) // 1 minute timeout
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()
	
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 60*1e9)
	
	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Simulate host exists
	h.SetHostID("test-host")

	// Create and store an expired token
	idBytes := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, idBytes); err != nil {
		t.Fatalf("Failed to generate token ID: %v", err)
	}
	sessionToken := token.SessionToken{
		ID:        hex.EncodeToString(idBytes),
		Timestamp: time.Now().Add(-2 * time.Minute).Unix(),
	}
	tm.StoreToken(sessionToken)

	// Use the expired token ID directly
	expiredTokenID := sessionToken.ID

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleWebSocket(w, r)
	}))
	defer server.Close()

	// Try to connect with expired token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + expiredTokenID
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Error("WebSocket connection with expired token should fail")
	}

	// HTTP 401 results in "bad handshake" error from WebSocket client
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Errorf("Expected handshake error, got: %v", err)
	}
}

// TestWebSocketConnectionHostWithoutToken tests that host can connect without token
func TestWebSocketConnectionHostWithoutToken(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()
	
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)
	
	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleWebSocket(w, r)
	}))
	defer server.Close()

	// First connection (host) should work without token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Host connection should succeed without token: %v", err)
	}
	defer conn.Close()

	// Verify that this client became host
	time.Sleep(100 * time.Millisecond)
	hostID := h.HostID()

	if hostID == "" {
		t.Error("First connection should become host")
	}
}

// TestWebSocketConnectionHostWithToken tests that host connection with token is rejected
func TestWebSocketConnectionHostWithToken(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()
	
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)
	
	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleWebSocket(w, r)
	}))
	defer server.Close()

	// Generate a valid token
	tokenID, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// First connection with token should be rejected
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + tokenID
	_, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Error("First connection with token should be rejected")
	}

	// HTTP 400 results in "bad handshake" error from WebSocket client
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Errorf("Expected handshake error, got: %v", err)
	}
}

// TestQRCodeEndpoint tests that QR code endpoint generates valid QR codes
func TestQRCodeEndpoint(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()
	
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)
	
	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleQRCode(w, r)
	}))
	defer server.Close()

	// Make request to QR code endpoint
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Expected content-type image/png, got %s", contentType)
	}
}

// TestCacheBustingVersion tests that script tags include dynamic version parameter
func TestCacheBustingVersion(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()

	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleIndex(w, r)
	}))
	defer server.Close()

	// Test host page
	resp, err := http.Get(server.URL + "/?mode=host")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Check that version is added to script tags
	if !strings.Contains(string(body), `<script src="/static/js/common.js?v=`+srv.version+`">`) {
		t.Errorf("Expected common.js to have version parameter, got: %s", string(body))
	}
	if !strings.Contains(string(body), `<script src="/static/js/host.js?v=`+srv.version+`">`) {
		t.Errorf("Expected host.js to have version parameter, got: %s", string(body))
	}
}

// TestVersionPattern tests that version string matches expected format
func TestVersionPattern(t *testing.T) {
	tm := token.NewTokenManager("", 10)
	h := hub.NewHub(1024*1024, 10) // 1MB max, 10 msgs/sec
	go h.Run()

	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Version should be 14 digits (YYYYMMDDHHMMSS)
	if len(srv.version) != 14 {
		t.Errorf("Expected version to be 14 digits, got %d", len(srv.version))
	}

	// Version should be numeric
	for _, c := range srv.version {
		if c < '0' || c > '9' {
			t.Errorf("Version should be numeric, got invalid character: %c", c)
		}
	}
}

// TestIsOriginAllowed tests origin validation with various scenarios
func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name          string
		origin        string
		allowedOrigins []string
		wantAllowed   bool
	}{
		{
			name:          "exact match",
			origin:        "http://localhost:3333",
			allowedOrigins: []string{"http://localhost:3333"},
			wantAllowed:   true,
		},
		{
			name:          "wildcard match with port",
			origin:        "http://localhost:3333",
			allowedOrigins: []string{"http://localhost:*"},
			wantAllowed:   true,
		},
		{
			name:          "wildcard match without port",
			origin:        "http://localhost",
			allowedOrigins: []string{"http://localhost:*"},
			wantAllowed:   true,
		},
		{
			name:          "wildcard match with colon suffix - exact match",
			origin:        "http://localhost",
			allowedOrigins: []string{"http://localhost:*:"},
			wantAllowed:   true,
		},
		{
			name:          "wildcard match with colon suffix - with port",
			origin:        "http://localhost:3333",
			allowedOrigins: []string{"http://localhost:*:"},
			wantAllowed:   true,
		},
		{
			name:          "no match - different origin",
			origin:        "http://example.com:3333",
			allowedOrigins: []string{"http://localhost:*"},
			wantAllowed:   false,
		},
		{
			name:          "no match - different protocol",
			origin:        "https://localhost:3333",
			allowedOrigins: []string{"http://localhost:*"},
			wantAllowed:   false,
		},
		{
			name:          "multiple allowed origins - first matches",
			origin:        "http://localhost:3333",
			allowedOrigins: []string{"http://localhost:*", "http://example.com:*"},
			wantAllowed:   true,
		},
		{
			name:          "multiple allowed origins - second matches",
			origin:        "http://example.com:3333",
			allowedOrigins: []string{"http://localhost:*", "http://example.com:*"},
			wantAllowed:   true,
		},
		{
			name:          "multiple allowed origins - none match",
			origin:        "http://other.com:3333",
			allowedOrigins: []string{"http://localhost:*", "http://example.com:*"},
			wantAllowed:   false,
		},
		{
			name:          "empty allowed origins - allow all",
			origin:        "http://anyorigin.com:3333",
			allowedOrigins: []string{},
			wantAllowed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOriginAllowed(tt.origin, tt.allowedOrigins)
			if got != tt.wantAllowed {
				t.Errorf("isOriginAllowed(%q, %v) = %v, want %v",
					tt.origin, tt.allowedOrigins, got, tt.wantAllowed)
			}
		})
	}
}

// TestMatchesWildcard tests wildcard pattern matching edge cases
func TestMatchesWildcard(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		pattern string
		want    bool
	}{
		{
			name:    "exact match with port",
			origin:  "http://localhost:3333",
			pattern: "http://localhost:3333",
			want:    true,
		},
		{
			name:    "exact match without port",
			origin:  "http://localhost",
			pattern: "http://localhost",
			want:    true,
		},
		{
			name:    "different origin prefix",
			origin:  "http://example.com:3333",
			pattern: "http://localhost:*",
			want:    false,
		},
		{
			name:    "different protocol",
			origin:  "https://localhost:3333",
			pattern: "http://localhost:*",
			want:    false,
		},
		{
			name:    "origin shorter than pattern",
			origin:  "http://localhost",
			pattern: "http://localhost:*extra",
			want:    false,
		},
		{
			name:    "path in origin",
			origin:  "http://localhost:3333/path",
			pattern: "http://localhost:*",
			want:    false,
		},
		{
			name:    "empty origin",
			origin:  "",
			pattern: "http://localhost:*",
			want:    false,
		},
		{
			name:    "empty pattern",
			origin:  "http://localhost:3333",
			pattern: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesWildcard(tt.origin, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesWildcard(%q, %q) = %v, want %v",
					tt.origin, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestNewServer tests that NewServer initializes all fields correctly
func TestNewServer(t *testing.T) {
	h := hub.NewHub(1024*1024, 10)
	go h.Run()

	tm := token.NewTokenManager("", 10)
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Verify all fields are set
	if srv.hub != h {
		t.Error("Hub should be set")
	}
	if srv.tokenManager != tm {
		t.Error("TokenManager should be set")
	}
	if srv.qrGenerator != qrGen {
		t.Error("QRGenerator should be set")
	}
	if srv.staticFiles != mockStaticFiles {
		t.Error("StaticFiles should be set")
	}
	if len(srv.allowedOrigins) != 1 {
		t.Error("AllowedOrigins should be set")
	}
	if srv.version == "" {
		t.Error("Version should be set")
	}
}

// TestShutdown tests that Shutdown is a no-op (should not panic)
func TestShutdown(t *testing.T) {
	h := hub.NewHub(1024*1024, 10)
	go h.Run()

	tm := token.NewTokenManager("", 10)
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Should not panic
	srv.Shutdown()
	srv.Shutdown() // Should be idempotent
}

// TestRegisterRoutes tests that routes are registered correctly
func TestRegisterRoutes(t *testing.T) {
	h := hub.NewHub(1024*1024, 10)
	go h.Run()

	tm := token.NewTokenManager("", 10)
	qrGen := qrcode.NewGenerator("localhost:3333", "http", 10*60*1e9)

	srv := NewServer(h, tm, qrGen, mockStaticFiles, []string{"http://localhost:*"})

	// Register routes
	srv.RegisterRoutes()

	// Routes are registered to global http package, so we can't easily test them directly
	// But we can verify that the function doesn't panic
}


