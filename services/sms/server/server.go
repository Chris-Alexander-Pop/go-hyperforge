// Package server implements the sms service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms"
	smsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the sms service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"sms"`
	Port        string `env:"PORT" env-default:"8086"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the SMS HTTP API.
type Server struct {
	rest   *rest.Server
	sender sms.Sender
	cfg    Config
}

// New constructs the SMS HTTP server with a memory adapter.
func New(cfg Config) *Server {
	return NewWithSender(cfg, smsmemory.New())
}

// NewWithSender constructs the server with a custom sms.Sender (tests).
func NewWithSender(cfg Config, sender sms.Sender) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:   r,
		sender: sender,
		cfg:    cfg,
	}
	s.routes()
	return s
}

// Sender returns the underlying sms.Sender (tests may type-assert to memory).
func (s *Server) Sender() sms.Sender { return s.sender }

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
	e.POST("/v1/sms/send", s.send)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type sendRequest struct {
	To       string `json:"to"`
	Body     string `json:"body"`
	From     string `json:"from"`
	MediaURL string `json:"media_url"`
}

func (s *Server) send(c echo.Context) error {
	var req sendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.To) == "" {
		return errors.InvalidArgument("to is required", nil)
	}
	if strings.TrimSpace(req.Body) == "" {
		return errors.InvalidArgument("body is required", nil)
	}

	msg := &sms.Message{
		From:     req.From,
		To:       req.To,
		Body:     req.Body,
		MediaURL: req.MediaURL,
	}
	if err := s.sender.Send(c.Request().Context(), msg); err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, map[string]string{"status": "sent"})
}
