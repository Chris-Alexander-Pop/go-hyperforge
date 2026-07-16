// Package server implements the cachinglayer service HTTP API.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	cachememory "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the cachinglayer service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"cachinglayer"`
	Port        string `env:"PORT" env-default:"8144"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the cache HTTP API.
type Server struct {
	rest  *rest.Server
	cache cache.Cache
	cfg   Config
}

// New constructs the cachinglayer HTTP server with an in-memory cache.
func New(cfg Config) *Server {
	return NewWithCache(cfg, cachememory.New())
}

// NewWithCache constructs the server with a custom cache.Cache (tests).
func NewWithCache(cfg Config, c cache.Cache) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cache: c, cfg: cfg}
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
	e.POST("/v1/caches", s.set)
	e.GET("/v1/caches/:key", s.get)
	e.DELETE("/v1/caches/:key", s.delete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type setRequest struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	TTLSeconds int64       `json:"ttl_seconds,omitempty"`
}

func (s *Server) set(c echo.Context) error {
	var req setRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	var ttl time.Duration
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	if err := s.cache.Set(c.Request().Context(), req.Key, req.Value, ttl); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"key":         req.Key,
		"ttl_seconds": req.TTLSeconds,
	})
}

func (s *Server) get(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	var value interface{}
	if err := s.cache.Get(c.Request().Context(), key, &value); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"key":   key,
		"value": value,
	})
}

func (s *Server) delete(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	if err := s.cache.Delete(c.Request().Context(), key); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
