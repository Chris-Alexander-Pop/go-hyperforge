// Package server implements the logaggregator service HTTP API.
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

// Config is the logaggregator service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"logaggregator"`
	Port        string `env:"PORT" env-default:"8103"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// LogEntry is an ingested log line.
type LogEntry struct {
	ID        string            `json:"id"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Service   string            `json:"service,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Server wraps the logs HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	logs []LogEntry
}

// New constructs the logaggregator HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, logs: make([]LogEntry, 0)}
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
	e.POST("/v1/logs", s.ingest)
	e.GET("/v1/logs", s.list)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type ingestRequest struct {
	Level   string            `json:"level"`
	Message string            `json:"message"`
	Service string            `json:"service,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

func (s *Server) ingest(c echo.Context) error {
	var req ingestRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		return errors.InvalidArgument("message is required", nil)
	}
	level := strings.ToLower(strings.TrimSpace(req.Level))
	if level == "" {
		level = "info"
	}
	switch level {
	case "debug", "info", "warn", "warning", "error", "fatal":
	default:
		return errors.InvalidArgument("level must be debug, info, warn, error, or fatal", nil)
	}
	if level == "warning" {
		level = "warn"
	}
	entry := LogEntry{
		ID:        uuid.NewString(),
		Level:     level,
		Message:   msg,
		Service:   strings.TrimSpace(req.Service),
		Labels:    req.Labels,
		Timestamp: time.Now().UTC(),
	}
	s.mu.Lock()
	s.logs = append(s.logs, entry)
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, entry)
}

func (s *Server) list(c echo.Context) error {
	level := strings.ToLower(strings.TrimSpace(c.QueryParam("level")))
	if level == "warning" {
		level = "warn"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]LogEntry, 0)
	for _, e := range s.logs {
		if level != "" && e.Level != level {
			continue
		}
		out = append(out, e)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"logs": out})
}
