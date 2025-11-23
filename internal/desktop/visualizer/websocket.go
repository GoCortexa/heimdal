package visualizer

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// UpdateMessage represents a real-time update sent to WebSocket clients
type UpdateMessage struct {
	Type      string      `json:"type"` // "device", "traffic", "anomaly"
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan *UpdateMessage
	id   string
}

// WebSocketHub manages WebSocket connections and broadcasts updates
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan *UpdateMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for local dashboard
		// In production, this should be more restrictive
		return true
	},
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan *UpdateMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		running:    false,
		stopChan:   make(chan struct{}),
	}
}

// Run starts the WebSocket hub's main loop
func (h *WebSocketHub) Run() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	log.Println("[WebSocketHub] Starting...")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocketHub] Client %s registered (total: %d)", client.id, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[WebSocketHub] Client %s unregistered (total: %d)", client.id, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Add timestamp to message
			message.Timestamp = time.Now().Format("2006-01-02T15:04:05Z07:00")

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// Message sent successfully
				default:
					// Client's send buffer is full, close the connection
					close(client.send)
					delete(h.clients, client)
					log.Printf("[WebSocketHub] Client %s removed due to full buffer", client.id)
				}
			}
			h.mu.RUnlock()

		case <-h.stopChan:
			log.Println("[WebSocketHub] Stopping...")
			h.mu.Lock()
			// Close all client connections
			for client := range h.clients {
				close(client.send)
				client.conn.Close()
			}
			h.clients = make(map[*WebSocketClient]bool)
			h.running = false
			h.mu.Unlock()
			return
		}
	}
}

// Stop gracefully stops the WebSocket hub
func (h *WebSocketHub) Stop() {
	h.mu.RLock()
	if !h.running {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	close(h.stopChan)
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(message *UpdateMessage) {
	h.mu.RLock()
	if !h.running {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	select {
	case h.broadcast <- message:
		// Message queued for broadcast
	default:
		log.Println("[WebSocketHub] Warning: broadcast channel full, dropping message")
	}
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// handleWebSocket handles WebSocket connection requests
func (v *Visualizer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Visualizer] WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &WebSocketClient{
		hub:  v.wsHub,
		conn: conn,
		send: make(chan *UpdateMessage, 256),
		id:   r.RemoteAddr,
	}

	// Register client with hub
	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocketClient] Unexpected close error: %v", err)
			}
			break
		}
		// We don't expect messages from clients, just keep the connection alive
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write message as JSON
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			if err := json.NewEncoder(w).Encode(message); err != nil {
				log.Printf("[WebSocketClient] Error encoding message: %v", err)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
