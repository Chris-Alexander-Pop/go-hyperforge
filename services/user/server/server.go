// Package server implements the user service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/user/internal/store"
	"github.com/labstack/echo/v4"
)

const headerUserID = "X-User-ID"

// Config is the user service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"user"`
	Port        string `env:"PORT" env-default:"8082"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the user HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
}

// New constructs the user HTTP server with an in-memory profile store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, store.New())
}

// NewWithStore constructs the user HTTP server with a custom store (tests).
func NewWithStore(cfg Config, profiles *store.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: profiles}
	s.routes()
	return s
}

// Echo exposes the underlying Echo instance.
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/users", s.create)
	e.GET("/v1/users/me", s.me)
	e.GET("/v1/users/:id", s.getByID)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}

	p, err := s.store.Upsert(c.Request().Context(), store.Profile{
		ID:    req.ID,
		Email: req.Email,
		Name:  req.Name,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, p)
}

func (s *Server) me(c echo.Context) error {
	userID := c.Request().Header.Get(headerUserID)
	if userID == "" {
		return errors.Unauthorized("missing X-User-ID header", nil)
	}
	p, err := s.store.Get(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}

func (s *Server) getByID(c echo.Context) error {
	caller := c.Request().Header.Get(headerUserID)
	id := c.Param("id")
	if caller == "" {
		return errors.Unauthorized("missing X-User-ID header", nil)
	}
	p, err := s.store.Get(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}
