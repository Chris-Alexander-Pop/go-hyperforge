// Package server implements the contextmanager service HTTP API.
package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	llmmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the contextmanager service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"contextmanager"`
	Port        string `env:"PORT" env-default:"8098"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
	MaxMessages int    `env:"CONTEXT_MAX_MESSAGES" env-default:"100"`
}

// Session is a conversation context handle.
type Session struct {
	ID string `json:"id"`
}

// Server wraps the contextmanager HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config

	mu       sync.RWMutex
	sessions map[string]llmmemory.Memory
}

// New constructs the contextmanager HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:     r,
		cfg:      cfg,
		sessions: make(map[string]llmmemory.Memory),
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
	e.POST("/v1/contexts", s.create)
	e.POST("/v1/contexts/:id/messages", s.appendMessage)
	e.GET("/v1/contexts/:id/messages", s.listMessages)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) create(c echo.Context) error {
	maxLen := s.cfg.MaxMessages
	if maxLen < 0 {
		maxLen = 0
	}
	id := uuid.NewString()
	mem := llmmemory.NewSimpleMemory(maxLen)

	s.mu.Lock()
	s.sessions[id] = mem
	s.mu.Unlock()

	return c.JSON(http.StatusCreated, Session{ID: id})
}

type appendMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *Server) appendMessage(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}

	var req appendMessageRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Content == "" {
		return errors.InvalidArgument("content is required", nil)
	}
	role := llm.Role(req.Role)
	if role == "" {
		role = llm.RoleUser
	}

	s.mu.RLock()
	mem, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("context not found", nil)
	}

	msg := llm.Message{Role: role, Content: req.Content}
	if err := mem.AddMessage(c.Request().Context(), msg); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, msg)
}

func (s *Server) listMessages(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}

	s.mu.RLock()
	mem, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("context not found", nil)
	}

	msgs, err := mem.GetMessages(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, msgs)
}
