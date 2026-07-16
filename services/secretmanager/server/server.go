// Package server implements the secretmanager service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	secretsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets/adapters/memory"
	"github.com/labstack/echo/v4"
)

// Config is the secretmanager service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"secretmanager"`
	Port        string `env:"PORT" env-default:"8108"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// deleter is implemented by adapters that support secret deletion.
type deleter interface {
	Delete(ctx context.Context, name string) error
}

// Server wraps the secrets HTTP API.
type Server struct {
	rest    *rest.Server
	manager secrets.SecretManager
	cfg     Config
}

// New constructs the secretmanager HTTP server with an in-memory secrets backend.
func New(cfg Config) *Server {
	return NewWithManager(cfg, secretsmemory.New())
}

// NewWithManager constructs the server with a custom SecretManager (tests).
func NewWithManager(cfg Config, manager secrets.SecretManager) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, manager: manager, cfg: cfg}
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
	e.POST("/v1/secrets", s.set)
	e.GET("/v1/secrets/:name", s.get)
	e.DELETE("/v1/secrets/:name", s.delete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func mapSecretsErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, secrets.ErrNotFound) {
		return errors.NotFound("secret not found", err)
	}
	if errors.Is(err, secrets.ErrInvalidArgument) {
		return errors.InvalidArgument("invalid secret argument", err)
	}
	return err
}

type setRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s *Server) set(c echo.Context) error {
	var req setRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	if err := mapSecretsErr(s.manager.Set(c.Request().Context(), req.Name, req.Value)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"name": req.Name, "status": "stored"})
}

func (s *Server) get(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	val, err := s.manager.Get(c.Request().Context(), name)
	if err != nil {
		return mapSecretsErr(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"name": name, "value": val})
}

func (s *Server) delete(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	d, ok := s.manager.(deleter)
	if !ok {
		return errors.Unimplemented("delete is not supported by this secrets backend", nil)
	}
	if err := mapSecretsErr(d.Delete(c.Request().Context(), name)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"name": name, "status": "deleted"})
}
