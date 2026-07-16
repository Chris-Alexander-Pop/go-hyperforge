// Package server implements the datacatalog service HTTP API.
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

// Config is the datacatalog service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"datacatalog"`
	Port        string `env:"PORT" env-default:"8140"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Dataset is a registered catalog dataset.
type Dataset struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Server wraps the catalogs HTTP API.
type Server struct {
	rest     *rest.Server
	cfg      Config
	mu       sync.RWMutex
	datasets map[string]Dataset
}

// New constructs the datacatalog HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, datasets: make(map[string]Dataset)}
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
	e.POST("/v1/catalogs", s.register)
	e.GET("/v1/catalogs", s.list)
	e.GET("/v1/catalogs/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	ds := Dataset{
		ID:          uuid.NewString(),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Tags:        req.Tags,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now().UTC(),
	}
	s.mu.Lock()
	s.datasets[ds.ID] = ds
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, ds)
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Dataset, 0, len(s.datasets))
	for _, d := range s.datasets {
		out = append(out, d)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"datasets": out})
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	ds, ok := s.datasets[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("dataset not found", nil)
	}
	return c.JSON(http.StatusOK, ds)
}
