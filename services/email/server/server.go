// Package server implements the email service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	emailmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the email service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"email"`
	Port        string `env:"PORT" env-default:"8085"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the email HTTP API.
type Server struct {
	rest   *rest.Server
	sender email.Sender
	cfg    Config
}

// New constructs the email HTTP server with a memory adapter.
func New(cfg Config) *Server {
	return NewWithSender(cfg, emailmemory.New())
}

// NewWithSender constructs the server with a custom email.Sender (tests).
func NewWithSender(cfg Config, sender email.Sender) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:   r,
		sender: sender,
		cfg:    cfg,
	}
	s.routes()
	return s
}

// Sender returns the underlying email.Sender (tests may type-assert to memory).
func (s *Server) Sender() email.Sender { return s.sender }

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
	e.POST("/v1/emails/send", s.send)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type sendRequest struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
	From    string   `json:"from"`
}

func (s *Server) send(c echo.Context) error {
	var req sendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if len(req.To) == 0 {
		return errors.InvalidArgument("to is required", nil)
	}
	if strings.TrimSpace(req.Subject) == "" {
		return errors.InvalidArgument("subject is required", nil)
	}
	if strings.TrimSpace(req.Text) == "" && strings.TrimSpace(req.HTML) == "" {
		return errors.InvalidArgument("text or html is required", nil)
	}

	msg := &email.Message{
		From:    req.From,
		To:      req.To,
		Subject: req.Subject,
		Body: email.Body{
			PlainText: req.Text,
			HTML:      req.HTML,
		},
	}
	if err := s.sender.Send(c.Request().Context(), msg); err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, map[string]string{"status": "sent"})
}
