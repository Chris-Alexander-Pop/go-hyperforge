package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/gorilla/websocket"
)

// Config configures WebSocket upgrade and origin checks.
type Config struct {
	// AllowedOrigins is the Origin allowlist. Use "*" to allow any origin.
	// An empty list rejects browser cross-origin requests (empty Origin is allowed
	// for non-browser clients).
	AllowedOrigins []string
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients map[*Client]bool

	Broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	mu     *concurrency.SmartRWMutex
	done   chan struct{}
	once   sync.Once
	config Config
}

func NewHub() *Hub {
	return NewHubWithConfig(Config{})
}

func NewHubWithConfig(cfg Config) *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "websocket-hub"}),
		done:       make(chan struct{}),
		config:     cfg,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			h.closeAllClients()
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.broadcast(message)
		}
	}
}

// Shutdown stops the hub loop and closes all client send channels.
// Safe to call multiple times.
func (h *Hub) Shutdown() {
	h.once.Do(func() {
		close(h.done)
	})
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}

// broadcast sends to all clients. Map mutation for stale clients uses a write lock
// (never mutates under RLock).
func (h *Hub) broadcast(message []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	var stale []*Client
	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			stale = append(stale, client)
		}
	}

	if len(stale) == 0 {
		return
	}

	h.mu.Lock()
	for _, client := range stale {
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
			close(client.send)
		}
	}
	h.mu.Unlock()
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func newUpgrader(cfg Config) websocket.Upgrader {
	allowAll := false
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = struct{}{}
	}

	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowAll {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Non-browser clients typically omit Origin.
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}
}

// ServeWs handles websocket requests from the peer using the hub's origin config.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader := newUpgrader(hub.config)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L().Error("websocket upgrade failed", "error", err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		logger.L().Error("failed to set read deadline", "error", err)
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			logger.L().Error("failed to set read deadline in pong handler", "error", err)
		}
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		select {
		case c.hub.Broadcast <- message:
		case <-c.hub.done:
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				logger.L().Error("failed to set write deadline", "error", err)
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logger.L().Error("failed to write close message", "error", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				logger.L().Error("failed to write message", "error", err)
				return
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				logger.L().Error("failed to set write deadline for ping", "error", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
