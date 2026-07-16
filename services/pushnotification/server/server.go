// Package server implements the pushnotification service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push"
	pushmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the pushnotification service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"pushnotification"`
	Port        string `env:"PORT" env-default:"8128"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the push notification HTTP API.
type Server struct {
	rest   *rest.Server
	sender push.Sender
	cfg    Config
}

// New constructs the pushnotification HTTP server with a memory adapter.
func New(cfg Config) *Server {
	return NewWithSender(cfg, pushmemory.New())
}

// NewWithSender constructs the server with a custom push.Sender (tests).
func NewWithSender(cfg Config, sender push.Sender) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:   r,
		sender: sender,
		cfg:    cfg,
	}
	s.routes()
	return s
}

// Sender returns the underlying push.Sender (tests may type-assert to memory).
func (s *Server) Sender() push.Sender { return s.sender }

// Echo exposes the underlying Echo instance (tests / custom mounts).
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.sender != nil {
		_ = s.sender.Close()
	}
	return s.rest.Shutdown(ctx)
}

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/pushes/send", s.send)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type sendRequest struct {
	Tokens   []string          `json:"tokens"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	ImageURL string            `json:"image_url"`
	Data     map[string]string `json:"data"`
	Platform string            `json:"platform"`
	Priority string            `json:"priority"`
	TTLSecs  int64             `json:"ttl_secs"`
}

func (s *Server) send(c echo.Context) error {
	var req sendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if len(req.Tokens) == 0 {
		return errors.InvalidArgument("tokens is required", nil)
	}
	if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.Body) == "" {
		return errors.InvalidArgument("title or body is required", nil)
	}

	msg := &push.Message{
		Tokens:   req.Tokens,
		Title:    req.Title,
		Body:     req.Body,
		ImageURL: req.ImageURL,
		Data:     req.Data,
		Platform: req.Platform,
		Priority: req.Priority,
	}
	if req.TTLSecs > 0 {
		msg.TTL = time.Duration(req.TTLSecs) * time.Second
	}
	if err := s.sender.Send(c.Request().Context(), msg); err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, map[string]string{"status": "sent"})
}
