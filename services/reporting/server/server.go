// Package server implements the reporting service HTTP API.
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

// Config is the reporting service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"reporting"`
	Port        string `env:"PORT" env-default:"8114"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// ReportJob is a report generation job.
type ReportJob struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	ResultURI string    `json:"result_uri,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Server wraps the reports HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	jobs map[string]*ReportJob
}

// New constructs the reporting HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, jobs: make(map[string]*ReportJob)}
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
	e.POST("/v1/reports", s.create)
	e.GET("/v1/reports/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
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
	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = "summary"
	}
	now := time.Now().UTC()
	job := &ReportJob{
		ID:        uuid.NewString(),
		Name:      name,
		Type:      typ,
		Status:    "completed",
		ResultURI: "memory://reports/" + uuid.NewString(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, job)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("report not found", nil)
	}
	return c.JSON(http.StatusOK, job)
}
