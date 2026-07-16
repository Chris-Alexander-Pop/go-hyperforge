// Package server implements the chat service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/chat"
	chatmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/chat/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the chat service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"chat"`
	Port        string `env:"PORT" env-default:"8129"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Room is a chat room.
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// StoredMessage is a persisted room message.
type StoredMessage struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Server wraps the chat HTTP API.
type Server struct {
	rest     *rest.Server
	sender   chat.Sender
	cfg      Config
	mu       sync.RWMutex
	rooms    map[string]Room
	messages map[string][]StoredMessage
}

// New constructs the chat HTTP server with an in-memory sender.
func New(cfg Config) *Server {
	return NewWithSender(cfg, chatmemory.New())
}

// NewWithSender constructs the server with a custom chat.Sender (tests).
func NewWithSender(cfg Config, sender chat.Sender) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:     r,
		sender:   sender,
		cfg:      cfg,
		rooms:    make(map[string]Room),
		messages: make(map[string][]StoredMessage),
	}
	s.routes()
	return s
}

// Echo exposes the underlying Echo instance (tests / custom mounts).
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/chats/rooms", s.createRoom)
	e.GET("/v1/chats/rooms", s.listRooms)
	e.POST("/v1/chats/rooms/:id/messages", s.postMessage)
	e.GET("/v1/chats/rooms/:id/messages", s.listMessages)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRoomRequest struct {
	Name string `json:"name"`
}

func (s *Server) createRoom(c echo.Context) error {
	var req createRoomRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	room := Room{ID: uuid.NewString(), Name: name, CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.rooms[room.ID] = room
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, room)
}

func (s *Server) listRooms(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		out = append(out, r)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"rooms": out})
}

type postMessageRequest struct {
	UserID string `json:"user_id"`
	Text   string `json:"text"`
}

func (s *Server) postMessage(c echo.Context) error {
	roomID := strings.TrimSpace(c.Param("id"))
	var req postMessageRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	userID := strings.TrimSpace(req.UserID)
	text := strings.TrimSpace(req.Text)
	if userID == "" {
		return errors.InvalidArgument("user_id is required", nil)
	}
	if text == "" {
		return errors.InvalidArgument("text is required", nil)
	}
	s.mu.RLock()
	_, ok := s.rooms[roomID]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("room not found", nil)
	}
	if err := s.sender.Send(c.Request().Context(), &chat.Message{
		ChannelID: roomID,
		UserID:    userID,
		Text:      text,
	}); err != nil {
		return err
	}
	msg := StoredMessage{
		ID:        uuid.NewString(),
		RoomID:    roomID,
		UserID:    userID,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.messages[roomID] = append(s.messages[roomID], msg)
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, msg)
}

func (s *Server) listMessages(c echo.Context) error {
	roomID := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.rooms[roomID]; !ok {
		return errors.NotFound("room not found", nil)
	}
	msgs := s.messages[roomID]
	out := make([]StoredMessage, len(msgs))
	copy(out, msgs)
	return c.JSON(http.StatusOK, map[string]interface{}{"messages": out})
}
