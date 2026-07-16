// Package server implements the dataretention service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/dataretention/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the dataretention service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"dataretention"`
	Port        string `env:"PORT" env-default:"8136"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the data retention HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the dataretention HTTP server with an in-memory store.
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
	e.POST("/v1/retention/policies", s.create)
	e.GET("/v1/retention/policies", s.list)
	e.POST("/v1/retention/evaluate", s.evaluate)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Resource string `json:"resource"`
	Days     int    `json:"days"`
}

type policyResponse struct {
	ID        string    `json:"id"`
	Resource  string    `json:"resource"`
	Days      int       `json:"days"`
	CreatedAt time.Time `json:"created_at"`
}

func toResponse(p *store.Policy) policyResponse {
	return policyResponse{
		ID:        p.ID,
		Resource:  p.Resource,
		Days:      p.Days,
		CreatedAt: p.CreatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	p, err := s.store.Create(c.Request().Context(), store.CreateInput{
		Resource: strings.TrimSpace(req.Resource),
		Days:     req.Days,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(p))
}

func (s *Server) list(c echo.Context) error {
	items, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]policyResponse, 0, len(items))
	for _, p := range items {
		out = append(out, toResponse(p))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"policies": out})
}

func (s *Server) evaluate(c echo.Context) error {
	result, err := s.store.Evaluate(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"policies_evaluated": result.PoliciesEvaluated,
		"expired_deleted":    result.ExpiredDeleted,
		"evaluated_at":       result.EvaluatedAt,
	})
}
