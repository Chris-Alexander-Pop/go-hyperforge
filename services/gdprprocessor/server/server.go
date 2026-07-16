// Package server implements the gdprprocessor service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/gdprprocessor/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the gdprprocessor service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"gdprprocessor"`
	Port        string `env:"PORT" env-default:"8137"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the GDPR processor HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the gdprprocessor HTTP server with an in-memory store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, store.New())
}

// NewWithStore constructs the server with a custom store (tests).
func NewWithStore(cfg Config, st *store.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: st, cfg: cfg}
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
	e.POST("/v1/gdpr/requests", s.create)
	e.GET("/v1/gdpr/requests/:id", s.get)
	e.POST("/v1/gdpr/requests/:id/complete", s.complete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Type      string `json:"type"`
	SubjectID string `json:"subject_id"`
}

type requestResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	SubjectID string    `json:"subject_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(r *store.Request) requestResponse {
	return requestResponse{
		ID:        r.ID,
		Type:      string(r.Type),
		SubjectID: r.SubjectID,
		Status:    string(r.Status),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	r, err := s.store.Create(c.Request().Context(), store.CreateInput{
		Type:      store.RequestType(strings.TrimSpace(req.Type)),
		SubjectID: strings.TrimSpace(req.SubjectID),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(r))
}

func (s *Server) get(c echo.Context) error {
	r, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(r))
}

func (s *Server) complete(c echo.Context) error {
	r, err := s.store.Complete(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(r))
}
