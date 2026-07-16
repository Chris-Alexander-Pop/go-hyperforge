// Package server implements the scheduledjobs service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/scheduler"
	"github.com/labstack/echo/v4"
)

// Config is the scheduledjobs service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"scheduledjobs"`
	Port        string `env:"PORT" env-default:"8118"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the jobs HTTP API backed by pkg/workflow/scheduler.
type Server struct {
	rest *rest.Server
	sch  *scheduler.Scheduler
	cfg  Config
	mu   sync.Mutex
	runs map[string]int
}

// New constructs the scheduledjobs HTTP server with an in-memory scheduler.
func New(cfg Config) *Server {
	sch := scheduler.New(scheduler.NewMemoryStore(), nil)
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, sch: sch, cfg: cfg, runs: make(map[string]int)}
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
	e.POST("/v1/jobs", s.create)
	e.GET("/v1/jobs", s.list)
	e.GET("/v1/jobs/:name", s.get)
	e.POST("/v1/jobs/:name/run", s.runOnce)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	schedule := strings.TrimSpace(req.Schedule)
	if schedule == "" {
		schedule = "once"
	}
	handler := func(ctx context.Context) error {
		s.mu.Lock()
		s.runs[name]++
		s.mu.Unlock()
		return nil
	}
	var err error
	if schedule == "once" {
		err = s.sch.ScheduleOnce(name, time.Now().UTC().Add(24*time.Hour), handler)
	} else {
		err = s.sch.Schedule(name, schedule, handler)
	}
	if err != nil {
		return err
	}
	job, err := s.sch.GetJob(name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, job)
}

func (s *Server) list(c echo.Context) error {
	jobs := s.sch.ListJobs()
	return c.JSON(http.StatusOK, map[string]interface{}{"jobs": jobs})
}

func (s *Server) get(c echo.Context) error {
	name := strings.TrimSpace(c.Param("name"))
	job, err := s.sch.GetJob(name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, job)
}

func (s *Server) runOnce(c echo.Context) error {
	name := strings.TrimSpace(c.Param("name"))
	exec, err := s.sch.RunNow(c.Request().Context(), name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	runs := s.runs[name]
	s.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"execution": exec,
		"runs":      runs,
	})
}
