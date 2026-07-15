package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/gorilla/websocket"
)

// AuthFunc authenticates a WebSocket upgrade request.
// Returning a non-nil error rejects the upgrade with HTTP 401.
// The returned context (or the request context if nil) is stored on the Client.
type AuthFunc func(r *http.Request) (ctx context.Context, err error)

// Config configures WebSocket upgrade, origin checks, and auth.
type Config struct {
	// AllowedOrigins is the Origin allowlist. Use "*" to allow any origin.
	// An empty list rejects browser cross-origin requests (empty Origin is allowed
	// for non-browser clients).
	AllowedOrigins []string

	// Authenticate is an optional upgrade-time auth hook. When set, ServeWs
	// invokes it before upgrading; failure returns 401 Unauthorized.
	Authenticate AuthFunc
}

// Hub maintains the set of active clients, rooms, and broadcasts messages.
type Hub struct {
	clients map[*Client]bool
	rooms   map[string]map[*Client]bool

	Broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	join  chan roomOp
	leave chan roomOp

	mu     *concurrency.SmartRWMutex
	done   chan struct{}
	once   sync.Once
	config Config
}

type roomOp struct {
	room   string
	client *Client
	done   chan struct{}
}

func NewHub() *Hub {
	return NewHubWithConfig(Config{})
}

func NewHubWithConfig(cfg Config) *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		join:       make(chan roomOp),
		leave:      make(chan roomOp),
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
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
			h.removeClient(client)
		case op := <-h.join:
			h.mu.Lock()
			members := h.rooms[op.room]
			if members == nil {
				members = make(map[*Client]bool)
				h.rooms[op.room] = members
			}
			members[op.client] = true
			op.client.rooms[op.room] = true
			h.mu.Unlock()
			if op.done != nil {
				close(op.done)
			}
		case op := <-h.leave:
			h.mu.Lock()
			if members, ok := h.rooms[op.room]; ok {
				delete(members, op.client)
				if len(members) == 0 {
					delete(h.rooms, op.room)
				}
			}
			delete(op.client.rooms, op.room)
			h.mu.Unlock()
			if op.done != nil {
				close(op.done)
			}
		case message := <-h.Broadcast:
			h.broadcast(message)
		}
	}
}

// JoinRoom adds client to a named room. Safe to call after the client is registered.
func (h *Hub) JoinRoom(client *Client, room string) {
	if client == nil || room == "" {
		return
	}
	done := make(chan struct{})
	select {
	case h.join <- roomOp{room: room, client: client, done: done}:
		<-done
	case <-h.done:
	}
}

// LeaveRoom removes client from a named room.
func (h *Hub) LeaveRoom(client *Client, room string) {
	if client == nil || room == "" {
		return
	}
	done := make(chan struct{})
	select {
	case h.leave <- roomOp{room: room, client: client, done: done}:
		<-done
	case <-h.done:
	}
}

// BroadcastToRoom sends a message to all clients in a room.
func (h *Hub) BroadcastToRoom(room string, message []byte) {
	h.mu.RLock()
	members := h.rooms[room]
	clients := make([]*Client, 0, len(members))
	for c := range members {
		clients = append(clients, c)
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
	for _, c := range stale {
		h.removeClient(c)
	}
}

// RoomSize returns the number of clients in a room.
func (h *Hub) RoomSize(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
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
	h.rooms = make(map[string]map[*Client]bool)
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	for room := range client.rooms {
		if members, ok := h.rooms[room]; ok {
			delete(members, client)
			if len(members) == 0 {
				delete(h.rooms, room)
			}
		}
	}
	client.rooms = nil
	close(client.send)
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

	for _, client := range stale {
		h.removeClient(client)
	}
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	ctx   context.Context
	rooms map[string]bool
}

// Context returns the authenticated/upgrade context for this client.
func (c *Client) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
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

// ServeWs handles websocket requests from the peer using the hub's origin config
// and optional upgrade-time Authenticate hook.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if hub.config.Authenticate != nil {
		authCtx, err := hub.config.Authenticate(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if authCtx != nil {
			ctx = authCtx
		}
	}

	upgrader := newUpgrader(hub.config)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L().Error("websocket upgrade failed", "error", err)
		return
	}
	client := &Client{
		hub:   hub,
		conn:  conn,
		send:  make(chan []byte, 256),
		ctx:   ctx,
		rooms: make(map[string]bool),
	}
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
