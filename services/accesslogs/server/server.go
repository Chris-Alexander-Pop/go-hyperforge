// Package server implements the accesslogs service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/audit"
	auditmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/audit/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the accesslogs service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"accesslogs"`
	Port        string `env:"PORT" env-default:"8138"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the access logs HTTP API backed by pkg/audit.
type Server struct {
	rest  *rest.Server
	store audit.Store
	cfg   Config
}

// New constructs the accesslogs HTTP server with an in-memory audit store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, auditmemory.NewStore())
}

// NewWithStore constructs the server with a custom audit.Store (tests).
func NewWithStore(cfg Config, store audit.Store) *Server {
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
	e.POST("/v1/access-logs", s.append)
	e.GET("/v1/access-logs", s.list)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type appendRequest struct {
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Outcome   string `json:"outcome"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

type entryResponse struct {
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource,omitempty"`
	Outcome   string    `json:"outcome"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func toResponse(e audit.Event) entryResponse {
	return entryResponse{
		UserID:    e.ActorID,
		Action:    e.Action,
		Resource:  e.ResourceID,
		Outcome:   string(e.Outcome),
		IP:        e.ActorIP,
		UserAgent: e.ActorUserAgent,
		Timestamp: e.Timestamp,
	}
}

func (s *Server) append(c echo.Context) error {
	var req appendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	userID := strings.TrimSpace(req.UserID)
	action := strings.TrimSpace(req.Action)
	if userID == "" {
		return errors.InvalidArgument("user_id is required", nil)
	}
	if action == "" {
		return errors.InvalidArgument("action is required", nil)
	}

	outcome := audit.Outcome(strings.TrimSpace(req.Outcome))
	if outcome == "" {
		outcome = audit.OutcomeSuccess
	}
	switch outcome {
	case audit.OutcomeSuccess, audit.OutcomeFailure, audit.OutcomeUnknown:
	default:
		return errors.InvalidArgument("outcome must be success, failure, or unknown", nil)
	}

	event := audit.Event{
		Timestamp:      time.Now().UTC(),
		EventType:      audit.EventTypeAccessGranted,
		Outcome:        outcome,
		ActorID:        userID,
		ActorType:      "user",
		ActorIP:        strings.TrimSpace(req.IP),
		ActorUserAgent: strings.TrimSpace(req.UserAgent),
		ResourceID:     strings.TrimSpace(req.Resource),
		Action:         action,
	}
	if outcome == audit.OutcomeFailure {
		event.EventType = audit.EventTypeAccessDenied
	}

	if err := s.store.Append(c.Request().Context(), event); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(event))
}

func (s *Server) list(c echo.Context) error {
	filter := audit.QueryFilter{
		ActorID: strings.TrimSpace(c.QueryParam("user_id")),
	}
	events, err := s.store.Query(c.Request().Context(), filter)
	if err != nil {
		return err
	}
	out := make([]entryResponse, 0, len(events))
	for _, e := range events {
		out = append(out, toResponse(e))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"entries": out})
}
