// Package server implements the llmgateway service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	llmmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the llmgateway service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"llmgateway"`
	Port        string `env:"PORT" env-default:"8095"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the LLM gateway HTTP API.
type Server struct {
	rest   *rest.Server
	client llm.Client
	cfg    Config
}

// New constructs the llmgateway HTTP server with an in-memory LLM client.
func New(cfg Config) *Server {
	return NewWithClient(cfg, llmmemory.New())
}

// NewWithClient constructs the server with a custom llm.Client (tests).
func NewWithClient(cfg Config, client llm.Client) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, client: client, cfg: cfg}
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
	e.POST("/v1/llm-requests/chat", s.chat)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type chatRequest struct {
	Messages []llm.Message `json:"messages"`
	Model    string        `json:"model,omitempty"`
}

func (s *Server) chat(c echo.Context) error {
	var req chatRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if len(req.Messages) == 0 {
		return errors.InvalidArgument("messages are required", nil)
	}

	var opts []llm.GenerateOption
	if req.Model != "" {
		opts = append(opts, llm.WithModel(req.Model))
	}

	gen, err := s.client.Chat(c.Request().Context(), req.Messages, opts...)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, gen)
}
