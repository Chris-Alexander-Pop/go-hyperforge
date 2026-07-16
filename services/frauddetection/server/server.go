// Package server implements the frauddetection service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/fraud"
	fraudmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/security/fraud/adapters/memory"
	"github.com/labstack/echo/v4"
)

// Config is the frauddetection service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"frauddetection"`
	Port        string `env:"PORT" env-default:"8131"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the fraud detection HTTP API.
type Server struct {
	rest     *rest.Server
	detector fraud.Detector
	cfg      Config
}

// New constructs the frauddetection HTTP server with an in-memory detector.
func New(cfg Config) *Server {
	return NewWithDetector(cfg, fraudmemory.New())
}

// NewWithDetector constructs the server with a custom Detector (tests).
func NewWithDetector(cfg Config, detector fraud.Detector) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, detector: detector, cfg: cfg}
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
	e.POST("/v1/fraud/score", s.score)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) score(c echo.Context) error {
	var event fraud.UserEvent
	if err := c.Bind(&event); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if event.UserID == "" && event.Action == "" {
		return errors.InvalidArgument("user_id or action is required", nil)
	}
	eval, err := s.detector.Score(c.Request().Context(), event)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, eval)
}
