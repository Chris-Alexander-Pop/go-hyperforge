// Package server implements the analytics service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	analyticsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the analytics service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"analytics"`
	Port        string `env:"PORT" env-default:"8113"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the analytics HTTP API.
type Server struct {
	rest   *rest.Server
	sink   analytics.Sink
	counts analytics.CounterStore
	cfg    Config
}

// New constructs the analytics HTTP server with in-memory adapters.
func New(cfg Config) *Server {
	return NewWithDeps(cfg, analyticsmemory.NewSink(), analyticsmemory.NewExact())
}

// NewWithDeps constructs the server with custom sink/counters (tests).
func NewWithDeps(cfg Config, sink analytics.Sink, counts analytics.CounterStore) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, sink: sink, counts: counts, cfg: cfg}
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
	e.POST("/v1/analytics/events", s.track)
	e.GET("/v1/analytics/counts", s.queryCounts)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type trackRequest struct {
	Name       string                 `json:"name"`
	UserID     string                 `json:"user_id,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

func (s *Server) track(c echo.Context) error {
	var req trackRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	props := map[string]any{}
	for k, v := range req.Properties {
		props[k] = v
	}
	ev := analytics.Event{
		Name:       name,
		UserID:     strings.TrimSpace(req.UserID),
		Properties: props,
		Timestamp:  time.Now().UTC(),
	}
	if err := s.sink.Ingest(c.Request().Context(), ev); err != nil {
		return err
	}
	if _, err := s.counts.Incr(c.Request().Context(), name, 1); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"name":    name,
		"user_id": ev.UserID,
		"tracked": true,
	})
}

func (s *Server) queryCounts(c echo.Context) error {
	name := strings.TrimSpace(c.QueryParam("name"))
	if name == "" {
		return errors.InvalidArgument("name query parameter is required", nil)
	}
	n, err := s.counts.Count(c.Request().Context(), name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"name": name, "count": n})
}
