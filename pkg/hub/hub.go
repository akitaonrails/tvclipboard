package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	ID           string
	Conn         *websocket.Conn
	Send         chan []byte
	Hub          *Hub
	Mobile       bool
	lastMessage  time.Time
	messageCount int
	mu           sync.Mutex
	closed       bool // Track if Send channel has been closed
}

// Hub manages all connected clients
type Hub struct {
	clients         map[string]*Client
	hostID          string
	broadcast       chan BroadcastMessage
	Register        chan *Client
	Unregister      chan *Client
	stop            chan struct{}
	mu              sync.RWMutex
	maxMessageSize  int64
	rateLimitPerSec int
}

// BroadcastMessage represents a message to broadcast to clients
type BroadcastMessage struct {
	Message []byte
	From    string // Don't send back to this client
}

// Message represents a WebSocket message
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	From    string `json:"from"`
	Role    string `json:"role,omitempty"`
}

// NewHub creates a new Hub
func NewHub(maxMessageSize int64, rateLimitPerSec int) *Hub {
	return &Hub{
		clients:         make(map[string]*Client),
		broadcast:       make(chan BroadcastMessage, 256),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		stop:            make(chan struct{}),
		mu:              sync.RWMutex{},
		maxMessageSize:  maxMessageSize,
		rateLimitPerSec: rateLimitPerSec,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
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
			msgBytes, err := json.Marshal(roleMsg)
			if err != nil {
				log.Printf("Failed to marshal role message: %v", err)
				h.mu.Unlock()
				continue
			}
			select {
			case client.Send <- msgBytes:
			default:
				log.Printf("Client %s send channel full, skipping role assignment", client.ID)
			}

			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				// Safely close the Send channel only if not already closed
				client.mu.Lock()
				if !client.closed {
					close(client.Send)
					client.closed = true
				}
				client.mu.Unlock()

				// If host disconnects, assign new host
				if client.ID == h.hostID {
					h.hostID = ""
					// Assign first remaining client as new host
					for id, c := range h.clients {
						h.hostID = id
						newHostMsg := Message{Type: "role", Role: "host"}
						msgBytes, err := json.Marshal(newHostMsg)
						if err != nil {
							log.Printf("Failed to marshal new host message: %v", err)
							continue
						}
						select {
						case c.Send <- msgBytes:
							log.Printf("Client %s promoted to HOST", id)
						default:
							log.Printf("Client %s send channel full, skipping host promotion", id)
						}
						break
					}
				}

				log.Printf("Client disconnected: %s", client.ID)
			}
			h.mu.Unlock()

		case broadcastMsg := <-h.broadcast:
			h.mu.Lock()
			for id, client := range h.clients {
				// Don't send back to the sender
				if id != broadcastMsg.From {
					select {
					case client.Send <- broadcastMsg.Message:
					default:
						log.Printf("Client %s send channel full, removing from hub", id)
						// Safely close the Send channel only if not already closed
						client.mu.Lock()
						if !client.closed {
							close(client.Send)
							client.closed = true
						}
						client.mu.Unlock()
						delete(h.clients, id)
					}
				}
			}
			h.mu.Unlock()

		case <-h.stop:
			// Stop signal received, exit the loop
			return
		}
	}
}

// Stop gracefully stops the hub
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case <-h.stop:
		// Already stopped
	default:
		close(h.stop)
	}
}

// checkRateLimit checks if client has exceeded rate limit using sliding window
func (c *Client) checkRateLimit(hub *Hub) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	timeSinceLast := now.Sub(c.lastMessage)

	// Reset count if more than a second has passed
	if timeSinceLast >= time.Second {
		c.messageCount = 1 // Count this message
		c.lastMessage = now
		return true
	}

	// Check if rate limit exceeded BEFORE incrementing
	if c.messageCount >= hub.rateLimitPerSec {
		log.Printf("Rate limit exceeded for client %s", c.ID)
		return false
	}

	c.messageCount++
	c.lastMessage = now // Update timestamp on each message to prevent burst attacks
	return true
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(c.Hub.maxMessageSize + 1024)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Check message size
		if int64(len(message)) > c.Hub.maxMessageSize {
			log.Printf("Message too large from %s: %d bytes (max: %d)", c.ID, len(message), c.Hub.maxMessageSize)
			errorMsg := Message{Type: "error", Content: fmt.Sprintf("Message too large. Maximum size is %d bytes.", c.Hub.maxMessageSize)}
			errorBytes, _ := json.Marshal(errorMsg)
			c.Conn.WriteMessage(websocket.TextMessage, errorBytes)
			continue
		}

		// Check rate limit
		if !c.checkRateLimit(c.Hub) {
			errorMsg := Message{Type: "error", Content: fmt.Sprintf("Rate limit exceeded. Maximum %d messages per second allowed.", c.Hub.rateLimitPerSec)}
			errorBytes, _ := json.Marshal(errorMsg)
			c.Conn.WriteMessage(websocket.TextMessage, errorBytes)
			continue
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			// Broadcast to all other clients (not back to sender)
			msg.From = c.ID
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message from %s: %v", c.ID, err)
				continue
			}
			broadcastMsg := BroadcastMessage{
				Message: msgBytes,
				From:    c.ID,
			}
			c.Hub.broadcast <- broadcastMsg
			log.Printf("Message from %s (type: %s, bytes: %d)", c.ID, msg.Type, len(msg.Content))
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Conn.Close()

	// Send periodic pings to detect dead connections
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WriteMessage error for client %s: %v", c.ID, err)
				return
			}
		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error for client %s: %v", c.ID, err)
				return
			}
		case <-c.Hub.stop:
			return
		}
	}
}

// HostID returns the current host's ID
func (h *Hub) HostID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hostID
}

// HasHost returns whether a host has been assigned
func (h *Hub) HasHost() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hostID != ""
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SetHostID sets the host ID (for testing only)
func (h *Hub) SetHostID(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hostID = id
}

// NewClient creates a new Client instance
func NewClient(conn *websocket.Conn, hub *Hub, mobile bool) *Client {
	return &Client{
		ID:           uuid.New().String(),
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Hub:          hub,
		Mobile:       mobile,
		lastMessage:  time.Now(),
		messageCount: 0,
	}
}
