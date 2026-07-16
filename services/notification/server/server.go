// Package server implements the notification orchestration HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	emailmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push"
	pushmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms"
	smsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the notification service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"notification"`
	Port        string `env:"PORT" env-default:"8084"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server orchestrates email, SMS, and push delivery.
type Server struct {
	rest  *rest.Server
	email email.Sender
	sms   sms.Sender
	push  push.Sender
	cfg   Config
}

// New constructs the notification HTTP server with in-memory channel senders.
func New(cfg Config) *Server {
	return NewWithSenders(cfg, emailmemory.New(), smsmemory.New(), pushmemory.New())
}

// NewWithSenders constructs the server with custom channel senders (tests).
func NewWithSenders(cfg Config, emailSender email.Sender, smsSender sms.Sender, pushSender push.Sender) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:  r,
		email: emailSender,
		sms:   smsSender,
		push:  pushSender,
		cfg:   cfg,
	}
	s.routes()
	return s
}

// EmailSender returns the email sender (tests may type-assert to memory).
func (s *Server) EmailSender() email.Sender { return s.email }

// SMSSender returns the SMS sender (tests may type-assert to memory).
func (s *Server) SMSSender() sms.Sender { return s.sms }

// PushSender returns the push sender (tests may type-assert to memory).
func (s *Server) PushSender() push.Sender { return s.push }

// Echo exposes the underlying Echo instance (tests / custom mounts).
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.email != nil {
		_ = s.email.Close()
	}
	if s.sms != nil {
		_ = s.sms.Close()
	}
	if s.push != nil {
		_ = s.push.Close()
	}
	return s.rest.Shutdown(ctx)
}

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/notifications/send", s.send)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type sendRequest struct {
	Channel string `json:"channel"`

	// Email fields
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
	From    string   `json:"from"`

	// SMS fields (ToPhone used when channel=sms; falls back to first To entry)
	ToPhone string `json:"to_phone"`
	Body    string `json:"body"`

	// Push fields
	Tokens []string          `json:"tokens"`
	Title  string            `json:"title"`
	Data   map[string]string `json:"data"`
}

func (s *Server) send(c echo.Context) error {
	var req sendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}

	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	ctx := c.Request().Context()

	switch channel {
	case "email":
		if len(req.To) == 0 {
			return errors.InvalidArgument("to is required for email", nil)
		}
		if strings.TrimSpace(req.Subject) == "" {
			return errors.InvalidArgument("subject is required for email", nil)
		}
		text := req.Text
		if text == "" {
			text = req.Body
		}
		if strings.TrimSpace(text) == "" && strings.TrimSpace(req.HTML) == "" {
			return errors.InvalidArgument("text or html is required for email", nil)
		}
		err := s.email.Send(ctx, &email.Message{
			From:    req.From,
			To:      req.To,
			Subject: req.Subject,
			Body:    email.Body{PlainText: text, HTML: req.HTML},
		})
		if err != nil {
			return err
		}
	case "sms":
		to := firstNonEmpty(req.ToPhone)
		if to == "" && len(req.To) > 0 {
			to = req.To[0]
		}
		if strings.TrimSpace(to) == "" {
			return errors.InvalidArgument("to or to_phone is required for sms", nil)
		}
		body := req.Body
		if body == "" {
			body = req.Text
		}
		if strings.TrimSpace(body) == "" {
			return errors.InvalidArgument("body is required for sms", nil)
		}
		if err := s.sms.Send(ctx, &sms.Message{From: req.From, To: to, Body: body}); err != nil {
			return err
		}
	case "push":
		if len(req.Tokens) == 0 {
			return errors.InvalidArgument("tokens is required for push", nil)
		}
		body := req.Body
		if body == "" {
			body = req.Text
		}
		if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(body) == "" {
			return errors.InvalidArgument("title or body is required for push", nil)
		}
		if err := s.push.Send(ctx, &push.Message{
			Tokens: req.Tokens,
			Title:  req.Title,
			Body:   body,
			Data:   req.Data,
		}); err != nil {
			return err
		}
	default:
		return errors.InvalidArgument("channel must be email, sms, or push", nil)
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"status":  "sent",
		"channel": channel,
	})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
