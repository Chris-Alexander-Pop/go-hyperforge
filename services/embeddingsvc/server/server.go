// Package server implements the embeddingsvc service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/nlp/embedding"
	embmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/nlp/embedding/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the embeddingsvc service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"embeddingsvc"`
	Port        string `env:"PORT" env-default:"8099"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
	Dimension   int    `env:"EMBEDDING_DIMENSION" env-default:"32"`
}

// Server wraps the embeddings HTTP API.
type Server struct {
	rest    *rest.Server
	service embedding.Service
	cfg     Config
}

// New constructs the embeddingsvc HTTP server with an in-memory embedding service.
func New(cfg Config) *Server {
	dim := cfg.Dimension
	if dim <= 0 {
		dim = 32
	}
	return NewWithService(cfg, embmemory.New(dim))
}

// NewWithService constructs the server with a custom embedding.Service (tests).
func NewWithService(cfg Config, svc embedding.Service) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, service: svc, cfg: cfg}
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
	e.POST("/v1/embeddings", s.embed)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type embedRequest struct {
	Texts []string `json:"texts"`
}

type embedResponse struct {
	Vectors   [][]float32 `json:"vectors"`
	Dimension int         `json:"dimension"`
}

func (s *Server) embed(c echo.Context) error {
	var req embedRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if len(req.Texts) == 0 {
		return errors.InvalidArgument("texts are required", nil)
	}

	vectors, err := s.service.Embed(c.Request().Context(), req.Texts)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, embedResponse{
		Vectors:   vectors,
		Dimension: s.service.Dimension(),
	})
}
