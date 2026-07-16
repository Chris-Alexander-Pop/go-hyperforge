// Package server implements the schemaregistry service HTTP API.
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

// Config is the schemaregistry service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"schemaregistry"`
	Port        string `env:"PORT" env-default:"8141"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// SchemaVersion is a registered schema revision.
type SchemaVersion struct {
	ID        string    `json:"id"`
	Subject   string    `json:"subject"`
	Version   int       `json:"version"`
	Format    string    `json:"format"`
	Schema    string    `json:"schema"`
	CreatedAt time.Time `json:"created_at"`
}

// Server wraps the schemas HTTP API.
type Server struct {
	rest     *rest.Server
	cfg      Config
	mu       sync.RWMutex
	byID     map[string]SchemaVersion
	versions map[string][]SchemaVersion // subject -> versions
}

// New constructs the schemaregistry HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, byID: make(map[string]SchemaVersion), versions: make(map[string][]SchemaVersion)}
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
	e.POST("/v1/schemas", s.register)
	e.GET("/v1/schemas/:id", s.get)
	e.GET("/v1/schemas", s.getBySubject)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Subject string `json:"subject"`
	Format  string `json:"format"`
	Schema  string `json:"schema"`
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	subject := strings.TrimSpace(req.Subject)
	schema := strings.TrimSpace(req.Schema)
	if subject == "" {
		return errors.InvalidArgument("subject is required", nil)
	}
	if schema == "" {
		return errors.InvalidArgument("schema is required", nil)
	}
	format := strings.TrimSpace(req.Format)
	if format == "" {
		format = "json"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ver := len(s.versions[subject]) + 1
	sv := SchemaVersion{
		ID:        uuid.NewString(),
		Subject:   subject,
		Version:   ver,
		Format:    format,
		Schema:    schema,
		CreatedAt: time.Now().UTC(),
	}
	s.byID[sv.ID] = sv
	s.versions[subject] = append(s.versions[subject], sv)
	return c.JSON(http.StatusCreated, sv)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	sv, ok := s.byID[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("schema not found", nil)
	}
	return c.JSON(http.StatusOK, sv)
}

func (s *Server) getBySubject(c echo.Context) error {
	subject := strings.TrimSpace(c.QueryParam("subject"))
	if subject == "" {
		return errors.InvalidArgument("subject query parameter is required", nil)
	}
	s.mu.RLock()
	vers := s.versions[subject]
	s.mu.RUnlock()
	if len(vers) == 0 {
		return errors.NotFound("schema subject not found", nil)
	}
	latest := vers[len(vers)-1]
	return c.JSON(http.StatusOK, map[string]interface{}{"subject": subject, "latest": latest, "versions": vers})
}
