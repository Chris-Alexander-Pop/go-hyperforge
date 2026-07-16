// Package server implements the backupsvc service HTTP API.
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

// Config is the backupsvc service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"backupsvc"`
	Port        string `env:"PORT" env-default:"8142"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Backup is a backup record.
type Backup struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	Status     string    `json:"status"`
	Location   string    `json:"location,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	RestoredAt time.Time `json:"restored_at,omitempty"`
}

// Server wraps the backups HTTP API.
type Server struct {
	rest    *rest.Server
	cfg     Config
	mu      sync.RWMutex
	backups map[string]*Backup
}

// New constructs the backupsvc HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, backups: make(map[string]*Backup)}
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
	e.POST("/v1/backups", s.create)
	e.GET("/v1/backups", s.list)
	e.POST("/v1/backups/:id/restore", s.restore)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Source   string `json:"source"`
	Location string `json:"location,omitempty"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		return errors.InvalidArgument("source is required", nil)
	}
	loc := strings.TrimSpace(req.Location)
	if loc == "" {
		loc = "memory://" + uuid.NewString()
	}
	b := &Backup{
		ID:        uuid.NewString(),
		Source:    source,
		Status:    "completed",
		Location:  loc,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.backups[b.ID] = b
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, b)
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Backup, 0, len(s.backups))
	for _, b := range s.backups {
		cp := *b
		out = append(out, &cp)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"backups": out})
}

func (s *Server) restore(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.backups[id]
	if !ok {
		return errors.NotFound("backup not found", nil)
	}
	b.Status = "restored"
	b.RestoredAt = time.Now().UTC()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"backup":  b,
		"message": "restore stub completed",
	})
}
