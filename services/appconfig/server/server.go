// Package server implements the appconfig service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the appconfig service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"appconfig"`
	Port        string `env:"PORT" env-default:"8092"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Entry is a config key/value.
type Entry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Server wraps the configs HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	kv   map[string]Entry
}

// New constructs the appconfig HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, kv: make(map[string]Entry)}
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
	e.PUT("/v1/configs/:key", s.set)
	e.POST("/v1/configs", s.setBody)
	e.GET("/v1/configs/:key", s.get)
	e.GET("/v1/configs", s.list)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type setRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (s *Server) setBody(c echo.Context) error {
	var req setRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	return s.upsert(c, req.Key, req.Value)
}

func (s *Server) set(c echo.Context) error {
	var req setRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	key := strings.TrimSpace(c.Param("key"))
	if strings.TrimSpace(req.Key) == "" {
		req.Key = key
	}
	return s.upsert(c, req.Key, req.Value)
}

func (s *Server) upsert(c echo.Context, key string, value interface{}) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	if value == nil {
		return errors.InvalidArgument("value is required", nil)
	}
	entry := Entry{Key: key, Value: value, UpdatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.kv[key] = entry
	s.mu.Unlock()
	return c.JSON(http.StatusOK, entry)
}

func (s *Server) get(c echo.Context) error {
	key := strings.TrimSpace(c.Param("key"))
	s.mu.RLock()
	entry, ok := s.kv[key]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("config key not found", nil)
	}
	return c.JSON(http.StatusOK, entry)
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.kv))
	for _, e := range s.kv {
		out = append(out, e)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"configs": out})
}
