package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
)

//go:embed static
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
	Mobile bool
}

type Hub struct {
	clients    map[string]*Client
	hostID     string
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	From    string `json:"from"`
	Role    string `json:"role,omitempty"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client

			// First client becomes host
			if h.hostID == "" {
				h.hostID = client.ID
				log.Printf("Client %s is now HOST (mobile: %v)", client.ID, client.Mobile)
			} else {
				log.Printf("Client connected: %s (mobile: %v)", client.ID, client.Mobile)
			}

			// Send role assignment to this client
			role := "client"
			if client.ID == h.hostID {
				role = "host"
			}
			roleMsg := Message{Type: "role", Role: role}
			msgBytes, _ := json.Marshal(roleMsg)
			client.Send <- msgBytes

			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)

				// If host disconnects, assign new host
				if client.ID == h.hostID {
					h.hostID = ""
					// Assign first remaining client as new host
					for id, c := range h.clients {
						h.hostID = id
						newHostMsg := Message{Type: "role", Role: "host"}
						msgBytes, _ := json.Marshal(newHostMsg)
						c.Send <- msgBytes
						log.Printf("Client %s promoted to HOST", id)
						break
					}
				}

				log.Printf("Client disconnected: %s", client.ID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			// Broadcast to all other clients
			msg.From = c.ID
			msgBytes, _ := json.Marshal(msg)
			c.Hub.broadcast <- msgBytes
			log.Printf("Message from %s: %s", c.ID, msg.Content)
		}
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func handleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	mobile := r.URL.Query().Get("mobile") == "true"
	client := &Client{
		ID:     uuid.New().String(),
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    hub,
		Mobile: mobile,
	}

	hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "localhost"
}

func main() {
	hub := NewHub()
	go hub.Run()

	port := "8080"
	localIP := getLocalIP()

	// QR code endpoint
	http.HandleFunc("/qrcode.png", func(w http.ResponseWriter, r *http.Request) {
		// Use the local IP address for the QR code
		host := localIP + ":" + port
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		url := scheme + "://" + host

		png, err := qrcode.Encode(url, qrcode.Medium, 256)
		if err != nil {
			http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
	})

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	// Serve static files
	staticContent, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal("Failed to create sub filesystem:", err)
	}
	fs := http.FileServer(http.FS(staticContent))
	http.Handle("/", fs)

	// Print helpful connection information
	log.Printf("Server starting on port %s\n", port)
	log.Printf("Local access: http://localhost:%s\n", port)
	if localIP != "localhost" {
		log.Printf("Network access: http://%s:%s\n", localIP, port)
		log.Printf("QR code will use: http://%s:%s\n", localIP, port)
	}
	log.Printf("Open in browser and scan QR code with your phone\n")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server error:", err)
	}
}
