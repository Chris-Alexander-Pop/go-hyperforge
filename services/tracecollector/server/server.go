// Package server implements the tracecollector service HTTP API.
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

// Config is the tracecollector service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"tracecollector"`
	Port        string `env:"PORT" env-default:"8104"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Span is a collected span within a trace.
type Span struct {
	ID        string            `json:"id"`
	TraceID   string            `json:"trace_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Service   string            `json:"service,omitempty"`
	Attrs     map[string]string `json:"attrs,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Server wraps the traces HTTP API.
type Server struct {
	rest  *rest.Server
	cfg   Config
	mu    sync.RWMutex
	spans map[string][]Span // trace_id -> spans
}

// New constructs the tracecollector HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, spans: make(map[string][]Span)}
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
	e.POST("/v1/traces", s.ingest)
	e.GET("/v1/traces/:trace_id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type ingestRequest struct {
	TraceID  string            `json:"trace_id"`
	ParentID string            `json:"parent_id,omitempty"`
	Name     string            `json:"name"`
	Service  string            `json:"service,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
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
	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = uuid.NewString()
	}
	span := Span{
		ID:        uuid.NewString(),
		TraceID:   traceID,
		ParentID:  strings.TrimSpace(req.ParentID),
		Name:      name,
		Service:   strings.TrimSpace(req.Service),
		Attrs:     req.Attrs,
		Timestamp: time.Now().UTC(),
	}
	s.mu.Lock()
	s.spans[traceID] = append(s.spans[traceID], span)
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, span)
}

func (s *Server) get(c echo.Context) error {
	traceID := strings.TrimSpace(c.Param("trace_id"))
	if traceID == "" {
		return errors.InvalidArgument("trace_id is required", nil)
	}
	s.mu.RLock()
	spans, ok := s.spans[traceID]
	s.mu.RUnlock()
	if !ok || len(spans) == 0 {
		return errors.NotFound("trace not found", nil)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"trace_id": traceID, "spans": spans})
}
