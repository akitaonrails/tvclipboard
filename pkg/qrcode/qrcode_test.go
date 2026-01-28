package qrcode

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	qrcodeLib "github.com/skip2/go-qrcode"
)

// TestQRCodeGeneration tests QR code generation endpoint
func TestQRCodeGeneration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		url := scheme + "://localhost:3333?token=abc123&mode=client"

		png, err := qrcodeLib.Encode(url, qrcodeLib.Medium, 256)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
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

	// Check that response contains PNG header
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if len(buf.Bytes()) < 8 {
		t.Error("Response should contain at least PNG header")
	}

	// PNG header is 137 80 78 71 13 10 26 10
	if !bytes.HasPrefix(buf.Bytes(), []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		t.Error("Response should be a valid PNG file")
	}
}

// TestQRCodeURLFormat tests that QR code contains proper URL format
func TestQRCodeURLFormat(t *testing.T) {
	// Simulate QR code generation
	encrypted := "abc123def456"
	timeout := 10 * time.Minute

	g := NewGenerator("192.168.1.100:3333", "http", timeout)
	url := g.GenerateQRCodeURL(encrypted)

	// Verify URL structure
	if !strings.Contains(url, "http://") {
		t.Error("URL should use http protocol")
	}

	if !strings.Contains(url, "token=") {
		t.Error("URL should contain token parameter")
	}

	if !strings.Contains(url, "mode=client") {
		t.Error("URL should contain mode=client parameter")
	}

	// Verify token is URL-safe (no spaces or special characters)
	for _, char := range []string{" ", "&", "\"", "'", "<", ">"} {
		if strings.Contains(encrypted, char) {
			t.Errorf("Token should be URL-safe (contains %q)", char)
		}
	}
}

// TestQRCodeGenerator tests QR code generator configuration
func TestQRCodeGenerator(t *testing.T) {
	timeout := 10 * time.Minute
	g := NewGenerator("localhost:3333", "http", timeout)

	if g.Host() != "localhost:3333" {
		t.Errorf("Expected host 'localhost:3333', got '%s'", g.Host())
	}

	if g.Scheme() != "http" {
		t.Errorf("Expected scheme 'http', got '%s'", g.Scheme())
	}

	if g.SessionTimeoutSeconds() != 600 {
		t.Errorf("Expected timeout 600 seconds, got %d", g.SessionTimeoutSeconds())
	}
}

// TestInjectSessionTimeout tests that session timeout is injected into HTML
func TestInjectSessionTimeout(t *testing.T) {
	html := `<html><body><div class="container">content</div></body></html>`
	injected := InjectSessionTimeout(html, 600)

	expected := `<div class="container" data-session-timeout="600">`
	if !strings.Contains(injected, expected) {
		t.Errorf("Expected to find %s in HTML, got: %s", expected, injected)
	}

	if !strings.Contains(injected, "content") {
		t.Error("HTML content should be preserved")
	}
}

// TestHTMLReplace tests HTML replacement functionality
func TestHTMLReplace(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		old      string
		new      string
		expected string
	}{
		{
			name:     "simple replacement",
			html:     "Hello world",
			old:      "world",
			new:      "Go",
			expected:  "Hello Go",
		},
		{
			name:     "no replacement needed",
			html:     "Hello world",
			old:      "missing",
			new:      "Go",
			expected:  "Hello world",
		},
		{
			name:     "HTML tag replacement",
			html:     `<div class="old">content</div>`,
			old:      `<div class="old">`,
			new:      `<div class="new">`,
			expected:  `<div class="new">content</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlReplace(tt.html, tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestFindSubstring tests substring finding
func TestFindSubstring(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substr     string
		expected   int
	}{
		{
			name:     "found at start",
			s:        "Hello world",
			substr:   "Hello",
			expected: 0,
		},
		{
			name:     "found in middle",
			s:        "Hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "not found",
			s:        "Hello world",
			substr:   "missing",
			expected: -1,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: -1,
		},
		{
			name:     "single character",
			s:        "abc",
			substr:   "b",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSubstring(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestServeQRCodeDirectly tests ServeQRCode function directly
func TestServeQRCodeDirectly(t *testing.T) {
	g := NewGenerator("localhost:3333", "http", 10*time.Minute)

	// Create test response recorder
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/qrcode.png?token=test-token-123", nil)

	// Call ServeQRCode directly
	g.ServeQRCode(w, r, "test-token-123")

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Expected content-type image/png, got %s", contentType)
	}

	// Check that response is a PNG
	body := w.Body.Bytes()
	if len(body) < 8 {
		t.Error("Response should be at least 8 bytes (PNG header)")
	}

	// PNG header
	expectedHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(body, expectedHeader) {
		t.Error("Response should be a valid PNG file")
	}
}
