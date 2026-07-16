// Package server implements the ratelimitersvc service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	apiratelimit "github.com/chris-alexander-pop/go-hyperforge/pkg/api/ratelimit"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	cachememory "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the ratelimitersvc service environment configuration.
type Config struct {
	ServiceName          string `env:"SERVICE_NAME" env-default:"ratelimitersvc"`
	Port                 string `env:"PORT" env-default:"8111"`
	LogLevel             string `env:"LOG_LEVEL" env-default:"info"`
	DefaultLimit         int64  `env:"RATE_LIMIT" env-default:"10"`
	DefaultPeriodSeconds int64  `env:"RATE_PERIOD_SECONDS" env-default:"60"`
}

// Server wraps the rate limit HTTP API.
type Server struct {
	rest    *rest.Server
	limiter apiratelimit.Limiter
	cfg     Config
}

// New constructs the ratelimitersvc HTTP server with an in-memory limiter.
func New(cfg Config) *Server {
	if cfg.DefaultLimit <= 0 {
		cfg.DefaultLimit = 10
	}
	if cfg.DefaultPeriodSeconds <= 0 {
		cfg.DefaultPeriodSeconds = 60
	}
	limiter := apiratelimit.New(cachememory.New(), apiratelimit.StrategyFixedWindow)
	return NewWithLimiter(cfg, limiter)
}

// NewWithLimiter constructs the server with a custom limiter (tests).
func NewWithLimiter(cfg Config, limiter apiratelimit.Limiter) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, limiter: limiter, cfg: cfg}
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
	e.POST("/v1/ratelimits/check", s.check)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type checkRequest struct {
	Key    string `json:"key"`
	Limit  int64  `json:"limit,omitempty"`
	Period int64  `json:"period_seconds,omitempty"`
}

func (s *Server) check(c echo.Context) error {
	var req checkRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = s.cfg.DefaultLimit
	}
	periodSec := req.Period
	if periodSec <= 0 {
		periodSec = s.cfg.DefaultPeriodSeconds
	}
	result, err := s.limiter.Allow(c.Request().Context(), key, limit, time.Duration(periodSec)*time.Second)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"key":       key,
		"allowed":   result.Allowed,
		"remaining": result.Remaining,
		"reset_ms":  result.Reset.Milliseconds(),
	})
}
