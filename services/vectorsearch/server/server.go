// Package server implements the vectorsearch service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	vectormemory "github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the vectorsearch service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"vectorsearch"`
	Port        string `env:"PORT" env-default:"8100"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the vectorsearch HTTP API.
type Server struct {
	rest  *rest.Server
	store vector.Store
	cfg   Config
}

// New constructs the vectorsearch HTTP server with an in-memory vector store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, vectormemory.New())
}

// NewWithStore constructs the server with a custom vector.Store (tests).
func NewWithStore(cfg Config, store vector.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: store, cfg: cfg}
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
	e.POST("/v1/vectors", s.upsert)
	e.POST("/v1/vectors/query", s.query)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type upsertRequest struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (s *Server) upsert(c echo.Context) error {
	var req upsertRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.ID == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	if len(req.Vector) == 0 {
		return errors.InvalidArgument("vector is required", nil)
	}

	if err := s.store.Upsert(c.Request().Context(), req.ID, req.Vector, req.Metadata); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"id": req.ID, "status": "upserted"})
}

type queryRequest struct {
	Vector []float32 `json:"vector"`
	TopK   int       `json:"top_k"`
}

type queryResponse struct {
	Matches []vector.Result `json:"matches"`
}

func (s *Server) query(c echo.Context) error {
	var req queryRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if len(req.Vector) == 0 {
		return errors.InvalidArgument("vector is required", nil)
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	matches, err := s.store.Search(c.Request().Context(), req.Vector, topK)
	if err != nil {
		return err
	}
	if matches == nil {
		matches = []vector.Result{}
	}
	return c.JSON(http.StatusOK, queryResponse{Matches: matches})
}
