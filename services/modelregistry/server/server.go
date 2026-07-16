// Package server implements the modelregistry service HTTP API.
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

// Config is the modelregistry service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"modelregistry"`
	Port        string `env:"PORT" env-default:"8121"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// ModelVersion is a registered model revision.
type ModelVersion struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Framework string            `json:"framework,omitempty"`
	URI       string            `json:"uri,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// Server wraps the models HTTP API.
type Server struct {
	rest   *rest.Server
	cfg    Config
	mu     sync.RWMutex
	byID   map[string]ModelVersion
	byName map[string][]ModelVersion
}

// New constructs the modelregistry HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, byID: make(map[string]ModelVersion), byName: make(map[string][]ModelVersion)}
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
	e.POST("/v1/models", s.register)
	e.GET("/v1/models/:id", s.get)
	e.GET("/v1/models", s.getByName)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Framework string            `json:"framework"`
	URI       string            `json:"uri"`
	Metadata  map[string]string `json:"metadata"`
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	version := strings.TrimSpace(req.Version)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	if version == "" {
		return errors.InvalidArgument("version is required", nil)
	}
	mv := ModelVersion{
		ID:        uuid.NewString(),
		Name:      name,
		Version:   version,
		Framework: strings.TrimSpace(req.Framework),
		URI:       strings.TrimSpace(req.URI),
		Metadata:  req.Metadata,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.byID[mv.ID] = mv
	s.byName[name] = append(s.byName[name], mv)
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, mv)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	mv, ok := s.byID[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("model not found", nil)
	}
	return c.JSON(http.StatusOK, mv)
}

func (s *Server) getByName(c echo.Context) error {
	name := strings.TrimSpace(c.QueryParam("name"))
	if name == "" {
		return errors.InvalidArgument("name query parameter is required", nil)
	}
	s.mu.RLock()
	vers := s.byName[name]
	s.mu.RUnlock()
	if len(vers) == 0 {
		return errors.NotFound("model not found", nil)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"name": name, "versions": vers, "latest": vers[len(vers)-1]})
}
