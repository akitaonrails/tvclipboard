package server

import (
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/websocket"
	"tvclipboard/pkg/hub"
	"tvclipboard/pkg/qrcode"
	"tvclipboard/pkg/token"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Server handles HTTP requests and WebSocket connections
type Server struct {
	hub            *hub.Hub
	tokenManager   *token.TokenManager
	qrGenerator    *qrcode.Generator
	staticFiles    fs.FS
	version        string
}

// NewServer creates a new Server instance
func NewServer(h *hub.Hub, tm *token.TokenManager, qrGen *qrcode.Generator, staticFiles fs.FS) *Server {
	return &Server{
		hub:          h,
		tokenManager: tm,
		qrGenerator:  qrGen,
		staticFiles:  staticFiles,
		version:      time.Now().Format("20060102150405"),
	}
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes() {
	// Main page handler
	http.HandleFunc("/", s.handleIndex)

	// QR code endpoint
	http.HandleFunc("/qrcode.png", s.handleQRCode)

	// WebSocket endpoint
	http.HandleFunc("/ws", s.handleWebSocket)

	// Serve static files (CSS, JS)
	staticContent, err := fs.Sub(s.staticFiles, "static")
	if err != nil {
		log.Fatal("Failed to create sub filesystem:", err)
	}
	fileServer := http.FileServer(http.FS(staticContent))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))
}

// handleIndex serves the host or client HTML page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")

	var templateFile string
	if mode == "client" {
		templateFile = "client.html"
	} else {
		templateFile = "host.html"
	}

	// Read and serve the template
	content, err := fs.ReadFile(s.staticFiles, "static/"+templateFile)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Inject session timeout as a data attribute and cache busting version
	htmlContent := string(content)
	if mode == "client" {
		htmlContent = qrcode.InjectSessionTimeout(htmlContent, s.qrGenerator.SessionTimeoutSeconds())
	}

	// Add version to all static JS files
	jsRegex := regexp.MustCompile(`(<script src="/static/js/[^"]+\.js)">`)
	htmlContent = jsRegex.ReplaceAllString(htmlContent, `$1?v=`+s.version+`">`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

// handleQRCode generates and serves a QR code with an encrypted token
func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request) {
	// Generate new session token
	encryptedToken, token, err := s.tokenManager.GenerateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated new session token: %s (expires in %v)", token.ID, s.tokenManager.Timeout())

	s.qrGenerator.ServeQRCode(w, r, encryptedToken)
}

// handleWebSocket handles WebSocket connection upgrades
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	hostExists := s.hub.HasHost()

	// Require token for client connections (when host already exists)
	if hostExists {
		if token == "" {
			log.Printf("Connection rejected: no token provided (host exists)")
			http.Error(w, "Token required for connection", http.StatusUnauthorized)
			return
		}

		_, err := s.tokenManager.ValidateToken(token)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
	} else if token != "" {
		// First connection (host) shouldn't have a token
		log.Printf("Connection rejected: token provided for first connection")
		http.Error(w, "Invalid connection - first connection should be from host page", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	mobile := r.URL.Query().Get("mobile") == "true"
	client := hub.NewClient(conn, s.hub, mobile)

	s.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
