// Package server implements the feedback service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the feedback service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"feedback"`
	Port        string `env:"PORT" env-default:"8126"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Entry is a submitted feedback record.
type Entry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	Target    string    `json:"target,omitempty"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Server wraps the feedback HTTP API.
type Server struct {
	rest    *rest.Server
	cfg     Config
	mu      sync.RWMutex
	entries []Entry
}

// New constructs the feedback HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, entries: make([]Entry, 0)}
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
	e.POST("/v1/feedback", s.submit)
	e.GET("/v1/feedback", s.list)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type submitRequest struct {
	UserID  string `json:"user_id"`
	Target  string `json:"target"`
	Rating  *int   `json:"rating"`
	Comment string `json:"comment"`
}

func (s *Server) submit(c echo.Context) error {
	var req submitRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Rating == nil {
		return errors.InvalidArgument("rating is required", nil)
	}
	rating := *req.Rating
	if rating < 1 || rating > 5 {
		return errors.InvalidArgument("rating must be between 1 and 5", nil)
	}
	comment := strings.TrimSpace(req.Comment)
	if comment == "" && strings.TrimSpace(req.Target) == "" {
		return errors.InvalidArgument("comment or target is required", nil)
	}
	entry := Entry{
		ID:        uuid.NewString(),
		UserID:    strings.TrimSpace(req.UserID),
		Target:    strings.TrimSpace(req.Target),
		Rating:    rating,
		Comment:   comment,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.entries = append(s.entries, entry)
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, entry)
}

func (s *Server) list(c echo.Context) error {
	target := strings.TrimSpace(c.QueryParam("target"))
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range s.entries {
		if target != "" && e.Target != target {
			continue
		}
		out = append(out, e)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"feedback": out})
}
