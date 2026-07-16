// Package server implements the featureflag service HTTP API.
package server

import (
	"context"
	"hash/fnv"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the featureflag service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"featureflag"`
	Port        string `env:"PORT" env-default:"8107"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Flag is a feature flag definition.
type Flag struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Enabled     bool      `json:"enabled"`
	Percentage  int       `json:"percentage,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Server wraps the feature flags HTTP API.
type Server struct {
	rest  *rest.Server
	cfg   Config
	mu    sync.RWMutex
	flags map[string]Flag // key -> flag
}

// New constructs the featureflag HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, flags: make(map[string]Flag)}
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
	e.POST("/v1/flags", s.create)
	e.POST("/v1/flags/evaluate", s.evaluate)
	e.GET("/v1/flags/:key", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Percentage  int    `json:"percentage,omitempty"`
	Description string `json:"description,omitempty"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	pct := req.Percentage
	if pct < 0 || pct > 100 {
		return errors.InvalidArgument("percentage must be between 0 and 100", nil)
	}
	flag := Flag{
		ID:          uuid.NewString(),
		Key:         key,
		Enabled:     req.Enabled,
		Percentage:  pct,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   time.Now().UTC(),
	}
	s.mu.Lock()
	s.flags[key] = flag
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, flag)
}

type evaluateRequest struct {
	Flag   string `json:"flag"`
	UserID string `json:"user_id"`
}

func (s *Server) evaluate(c echo.Context) error {
	var req evaluateRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	key := strings.TrimSpace(req.Flag)
	userID := strings.TrimSpace(req.UserID)
	if key == "" {
		return errors.InvalidArgument("flag is required", nil)
	}
	if userID == "" {
		return errors.InvalidArgument("user_id is required", nil)
	}
	s.mu.RLock()
	flag, ok := s.flags[key]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("flag not found", nil)
	}
	enabled := false
	if flag.Enabled {
		switch {
		case flag.Percentage <= 0 || flag.Percentage >= 100:
			enabled = true
		default:
			h := fnv.New32a()
			_, _ = h.Write([]byte(key + ":" + userID))
			enabled = int(h.Sum32()%100) < flag.Percentage
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"flag":    key,
		"user_id": userID,
		"enabled": enabled,
	})
}

func (s *Server) get(c echo.Context) error {
	key := strings.TrimSpace(c.Param("key"))
	s.mu.RLock()
	flag, ok := s.flags[key]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("flag not found", nil)
	}
	return c.JSON(http.StatusOK, flag)
}
