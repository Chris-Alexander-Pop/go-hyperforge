// Package server implements the metricscollector service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the metricscollector service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"metricscollector"`
	Port        string `env:"PORT" env-default:"8102"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Datapoint is an ingested metric sample.
type Datapoint struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Server wraps the metrics HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	pts  []Datapoint
}

// New constructs the metricscollector HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, pts: make([]Datapoint, 0)}
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
	e.POST("/v1/metrics", s.ingest)
	e.GET("/v1/metrics", s.query)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type ingestRequest struct {
	Name   string            `json:"name"`
	Value  *float64          `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}

func (s *Server) ingest(c echo.Context) error {
	var req ingestRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	if req.Value == nil {
		return errors.InvalidArgument("value is required", nil)
	}
	dp := Datapoint{
		ID:        uuid.NewString(),
		Name:      name,
		Value:     *req.Value,
		Labels:    req.Labels,
		Timestamp: time.Now().UTC(),
	}
	s.mu.Lock()
	s.pts = append(s.pts, dp)
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, dp)
}

func (s *Server) query(c echo.Context) error {
	name := strings.TrimSpace(c.QueryParam("name"))
	if name == "" {
		return errors.InvalidArgument("name query parameter is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Datapoint, 0)
	for _, p := range s.pts {
		if p.Name == name {
			out = append(out, p)
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"name": name, "datapoints": out})
}
