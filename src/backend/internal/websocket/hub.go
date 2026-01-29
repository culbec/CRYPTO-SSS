package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte
	ID   string
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan *Message
	register     chan *Client
	unregister   chan *Client
	done         chan struct{} // closed to signal Run() to stop
	stopped      chan struct{} // closed when Run() has returned
	shutdownOnce sync.Once
	mu           sync.RWMutex
	logger       *slog.Logger
}

// Message represents a WebSocket message
type Message struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// NewHub creates a new WebSocket hub
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
		stopped:    make(chan struct{}),
		logger:     logger,
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Run starts the hub's main loop. It returns when Shutdown is called.
func (h *Hub) Run() {
	defer close(h.stopped)
	for {
		select {
		case <-h.done:
			h.mu.Lock()
			for client := range h.clients {
				close(client.Send)
			}
			h.clients = make(map[*Client]bool)
			h.mu.Unlock()
			h.logger.Info("WebSocket hub stopped")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("WebSocket client connected", "id", client.ID, "total", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.logger.Info("WebSocket client disconnected", "id", client.ID, "total", len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.logger.Info("Broadcasting WebSocket event", "event", message.Event, "clients", len(h.clients))

			messageBytes, err := json.Marshal(message)
			if err != nil {
				h.logger.Error("Failed to marshal message", "error", err)
				continue
			}

			var toUnregister []*Client
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- messageBytes:
					// Message sent successfully
				default:
					toUnregister = append(toUnregister, client)
				}
			}
			h.mu.RUnlock()
			for _, client := range toUnregister {
				h.Unregister(client)
			}
			h.logger.Info("Broadcast complete", "event", message.Event)
		}
	}
}

// Emit broadcasts an event to all connected clients
func (h *Hub) Emit(event string, data interface{}) {
	h.broadcast <- &Message{
		Event: event,
		Data:  data,
	}
}

// Shutdown signals the hub to stop and blocks until Run has returned.
// All client Send channels are closed so WritePump goroutines exit.
// Idempotent: later calls block until the first shutdown completes.
func (h *Hub) Shutdown() {
	h.shutdownOnce.Do(func() {
		close(h.done)
		<-h.stopped
	})
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket read error", "error", err, "client", c.ID)
			}
			break
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			c.Hub.logger.Error("WebSocket write error", "error", err, "client", c.ID)
			return
		}
	}
	// Hub closed the channel
	c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
}
