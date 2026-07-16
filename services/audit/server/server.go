// Package server implements the audit service HTTP API.
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

// Config is the audit service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"audit"`
	Port        string `env:"PORT" env-default:"8093"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the audits HTTP API backed by pkg/audit.
type Server struct {
	rest  *rest.Server
	store audit.Store
	cfg   Config
}

// New constructs the audit HTTP server with an in-memory store.
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
	e.POST("/v1/audits", s.append)
	e.GET("/v1/audits", s.query)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type appendRequest struct {
	ActorID    string `json:"actor_id"`
	Action     string `json:"action"`
	ResourceID string `json:"resource_id"`
	EventType  string `json:"event_type"`
	Outcome    string `json:"outcome"`
}

func (s *Server) append(c echo.Context) error {
	var req appendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	actor := strings.TrimSpace(req.ActorID)
	action := strings.TrimSpace(req.Action)
	if actor == "" {
		return errors.InvalidArgument("actor_id is required", nil)
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
	et := audit.EventType(strings.TrimSpace(req.EventType))
	if et == "" {
		et = audit.EventTypeDataUpdate
	}
	event := audit.Event{
		Timestamp:  time.Now().UTC(),
		EventType:  et,
		Outcome:    outcome,
		ActorID:    actor,
		ActorType:  "user",
		ResourceID: strings.TrimSpace(req.ResourceID),
		Action:     action,
	}
	if err := s.store.Append(c.Request().Context(), event); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, event)
}

func (s *Server) query(c echo.Context) error {
	filter := audit.QueryFilter{
		ActorID: strings.TrimSpace(c.QueryParam("actor_id")),
	}
	if et := strings.TrimSpace(c.QueryParam("event_type")); et != "" {
		filter.EventType = audit.EventType(et)
	}
	events, err := s.store.Query(c.Request().Context(), filter)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"audits": events})
}
