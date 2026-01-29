package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"tvclipboard/i18n"
	"tvclipboard/pkg/hub"
	"tvclipboard/pkg/qrcode"
	"tvclipboard/pkg/token"
)

// Pre-compiled regexes for cache busting (compiled once at package init)
var (
	jsRegex  = regexp.MustCompile(`(<script src="/static/js/[^"]+\.js)">`)
	cssRegex = regexp.MustCompile(`(<link[^>]+href="/static/css/[^"]+\.css"[^>]*>)`)
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// isOriginAllowed checks if the given origin is in the allowed origins list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return true
	}
	for _, allowed := range allowedOrigins {
		// Exact match first (most common case)
		if origin == allowed {
			return true
		}
		// Check for wildcard at the end
		// Handle both "*" and "*:" patterns
		if len(allowed) > 0 && allowed[len(allowed)-1] == '*' {
			prefix := strings.TrimSuffix(allowed[:len(allowed)-1], ":")
			// Remove port from origin when checking against wildcard prefix
			// e.g., if allowed is "http://localhost:*" and origin is "http://localhost:3333",
			// extract "http://localhost" from both for comparison
			if matchesWildcard(origin, prefix) {
				return true
			}
		}
		// Check for "*:" pattern explicitly (when last char is ':')
		if len(allowed) > 1 && strings.HasSuffix(allowed, "*:") {
			prefix := allowed[:len(allowed)-2]
			// e.g., if allowed is "http://localhost:*:" and origin is "http://localhost:3333",
			// extract "http://localhost" from both for comparison
			if matchesWildcard(origin, prefix) {
				return true
			}
		}
	}
	return false
}

// matchesWildcard checks if origin matches a wildcard prefix
func matchesWildcard(origin, pattern string) bool {
	// Remove trailing colon from pattern if present (from patterns like "http://localhost:*")
	pattern = strings.TrimSuffix(pattern, ":")

	// Simple case: origin starts with pattern and either ends with port or equals pattern without :
	if len(origin) >= len(pattern) {
		originPrefix := origin[:len(pattern)]
		if originPrefix != pattern {
			return false
		}
		// If origin exactly matches pattern, allow it
		if len(origin) == len(pattern) {
			return true
		}
		// If origin is longer (has port), ensure next char is ':'
		if origin[len(pattern)] == ':' {
			return true
		}
	}
	return false
}

// setUpgraderOrigins configures the WebSocket upgrader with allowed origins
func setUpgraderOrigins(allowedOrigins []string) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// If allowedOrigins is configured, require a valid Origin header
		// Empty origin is only allowed when no origins are explicitly configured
		if origin == "" {
			if len(allowedOrigins) > 0 {
				log.Printf("Origin check failed: empty origin header with configured allowed origins")
				return false
			}
			return true // Allow requests without Origin header only when no restrictions configured
		}
		allowed := isOriginAllowed(origin, allowedOrigins)
		if !allowed {
			log.Printf("Origin check failed: %s not in allowed origins", origin)
		}
		return allowed
	}
}

// Server handles HTTP requests and WebSocket connections
type Server struct {
	hub            *hub.Hub
	tokenManager   *token.TokenManager
	qrGenerator    *qrcode.Generator
	staticFiles    fs.FS
	allowedOrigins []string
	version        string
	i18n           *i18n.I18n
}

// NewServer creates a new Server instance
func NewServer(h *hub.Hub, tm *token.TokenManager, qrGen *qrcode.Generator, staticFiles fs.FS, allowedOrigins []string, i18n *i18n.I18n) *Server {
	return &Server{
		hub:            h,
		tokenManager:   tm,
		qrGenerator:    qrGen,
		staticFiles:    staticFiles,
		allowedOrigins: allowedOrigins,
		version:        time.Now().Format("20060102150405"),
		i18n:           i18n,
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	// No-op: server shutdown is handled by http.Server.Shutdown()
}

// securityHeaders middleware adds security headers to all responses
func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; font-src https://cdnjs.cloudflare.com; img-src 'self' data:;")
		next(w, r)
	}
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes() {
	// Configure WebSocket upgrader with allowed origins
	setUpgraderOrigins(s.allowedOrigins)

	// Main page handler
	http.HandleFunc("/", securityHeaders(s.handleIndex))

	// QR code endpoint
	http.HandleFunc("/qrcode.png", s.handleQRCode)

	// WebSocket endpoint
	http.HandleFunc("/ws", s.handleWebSocket)

	// i18n endpoint
	http.HandleFunc("/i18n.json", s.handleI18n)

	// Serve static files (CSS, JS)
	staticContent, err := fs.Sub(s.staticFiles, "static")
	if err != nil {
		log.Printf("Failed to create sub filesystem: %v", err)
		return
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
	htmlContent = qrcode.InjectSessionTimeout(htmlContent, s.qrGenerator.SessionTimeoutSeconds())

	// Add version to all static JS files (using pre-compiled regex)
	htmlContent = jsRegex.ReplaceAllString(htmlContent, `$1?v=`+s.version+`">`)

	// Add version to all static CSS files (using pre-compiled regex)
	htmlContent = cssRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		return strings.Replace(match, ".css", `.css?v=`+s.version, 1)
	})

	// Add i18n script before body closing tag
	// Note: ToJSON() uses json.Marshal which properly escapes special characters
	i18nJSON, err := s.i18n.ToJSON()
	if err != nil {
		log.Printf("Failed to serialize i18n translations: %v", err)
		i18nJSON = []byte("{}")
	}

	// Inject translations as properly escaped JSON (json.Marshal handles escaping)
	safeJSON := strings.ReplaceAll(string(i18nJSON), "</", "<\\/")
	htmlContent = strings.Replace(htmlContent, "</body>", `<script>window.translations = `+safeJSON+`;</script></body>`, 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(htmlContent)); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

// handleI18n serves i18n translations as JSON
func (s *Server) handleI18n(w http.ResponseWriter, r *http.Request) {
	translations, err := s.i18n.GetTranslations()
	if err != nil {
		http.Error(w, "Failed to get translations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(translations); err != nil {
		log.Printf("Failed to encode i18n response: %v", err)
	}
}

// handleQRCode generates and serves a QR code with a session token
func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request) {
	// Generate new session token
	token, err := s.tokenManager.GenerateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated new session token (expires in %v)", s.tokenManager.Timeout())

	s.qrGenerator.ServeQRCode(w, r, token)
}

// handleWebSocket handles WebSocket connection upgrades
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	// Check origin before proceeding with WebSocket upgrade
	origin := r.Header.Get("Origin")
	if origin != "" {
		if !isOriginAllowed(origin, s.allowedOrigins) {
			log.Printf("Connection rejected: origin not allowed - %s", origin)
			http.Error(w, "Forbidden: Origin not allowed", http.StatusForbidden)
			return
		}
	}

	hostExists := s.hub.HasHost()

	// Log connection attempt without exposing the token value
	log.Printf("WebSocket connection attempt, hasToken: %v, hostExists: %v", token != "", hostExists)

	// Require token for client connections (when host already exists)
	if hostExists {
		if token == "" {
			log.Printf("Connection rejected: no token provided (host exists)")
			http.Error(w, "Unauthorized: valid token required", http.StatusUnauthorized)
			return
		}

		err := s.tokenManager.ValidateToken(token)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			http.Error(w, "Unauthorized: invalid or expired token", http.StatusUnauthorized)
			return
		}
	} else if token != "" {
		// First connection (host) shouldn't have a token
		log.Printf("Connection rejected: token provided for first connection")
		http.Error(w, "Bad request: first connection should not include token", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	log.Printf("WebSocket connection established")

	mobile := r.URL.Query().Get("mobile") == "true"
	client := hub.NewClient(conn, s.hub, mobile)

	s.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
