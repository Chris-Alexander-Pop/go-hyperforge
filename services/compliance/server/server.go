// Package server implements the compliance service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/compliance/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the compliance service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"compliance"`
	Port        string `env:"PORT" env-default:"8135"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the compliance HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the compliance HTTP server with an in-memory store.
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
	e.POST("/v1/compliance/checks", s.record)
	e.GET("/v1/compliance/checks", s.list)
	e.GET("/v1/compliance/checks/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type recordRequest struct {
	Policy    string `json:"policy"`
	SubjectID string `json:"subject_id"`
	Result    string `json:"result"`
}

type checkResponse struct {
	ID        string    `json:"id"`
	Policy    string    `json:"policy"`
	SubjectID string    `json:"subject_id"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"created_at"`
}

func toResponse(ch *store.Check) checkResponse {
	return checkResponse{
		ID:        ch.ID,
		Policy:    ch.Policy,
		SubjectID: ch.SubjectID,
		Result:    ch.Result,
		CreatedAt: ch.CreatedAt,
	}
}

func (s *Server) record(c echo.Context) error {
	var req recordRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	ch, err := s.store.Record(c.Request().Context(), store.RecordInput{
		Policy:    strings.TrimSpace(req.Policy),
		SubjectID: strings.TrimSpace(req.SubjectID),
		Result:    strings.TrimSpace(req.Result),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(ch))
}

func (s *Server) list(c echo.Context) error {
	items, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]checkResponse, 0, len(items))
	for _, ch := range items {
		out = append(out, toResponse(ch))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"checks": out})
}

func (s *Server) get(c echo.Context) error {
	ch, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(ch))
}
