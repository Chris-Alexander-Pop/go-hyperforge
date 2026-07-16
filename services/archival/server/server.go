// Package server implements the archival service HTTP API.
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

// Config is the archival service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"archival"`
	Port        string `env:"PORT" env-default:"8143"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// ArchiveObject is archived object metadata.
type ArchiveObject struct {
	ID         string            `json:"id"`
	ObjectKey  string            `json:"object_key"`
	SourceURI  string            `json:"source_uri,omitempty"`
	Tier       string            `json:"tier"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ArchivedAt time.Time         `json:"archived_at"`
}

// Server wraps the archives HTTP API.
type Server struct {
	rest     *rest.Server
	cfg      Config
	mu       sync.RWMutex
	archives map[string]ArchiveObject
}

// New constructs the archival HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, archives: make(map[string]ArchiveObject)}
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
	e.POST("/v1/archives", s.archive)
	e.GET("/v1/archives", s.list)
	e.GET("/v1/archives/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type archiveRequest struct {
	ObjectKey string            `json:"object_key"`
	SourceURI string            `json:"source_uri,omitempty"`
	Tier      string            `json:"tier,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func (s *Server) archive(c echo.Context) error {
	var req archiveRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	key := strings.TrimSpace(req.ObjectKey)
	if key == "" {
		return errors.InvalidArgument("object_key is required", nil)
	}
	tier := strings.TrimSpace(req.Tier)
	if tier == "" {
		tier = "cold"
	}
	obj := ArchiveObject{
		ID:         uuid.NewString(),
		ObjectKey:  key,
		SourceURI:  strings.TrimSpace(req.SourceURI),
		Tier:       tier,
		Metadata:   req.Metadata,
		ArchivedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.archives[obj.ID] = obj
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, obj)
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ArchiveObject, 0, len(s.archives))
	for _, a := range s.archives {
		out = append(out, a)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"archives": out})
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	obj, ok := s.archives[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("archive not found", nil)
	}
	return c.JSON(http.StatusOK, obj)
}
