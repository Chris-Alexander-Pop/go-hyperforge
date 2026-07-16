// Package server implements the finetuning service HTTP API.
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

// Config is the finetuning service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"finetuning"`
	Port        string `env:"PORT" env-default:"8120"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Job is a fine-tuning job.
type Job struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Dataset   string    `json:"dataset"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Server wraps the finetunes HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	jobs map[string]*Job
}

// New constructs the finetuning HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, jobs: make(map[string]*Job)}
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
	e.POST("/v1/finetunes", s.create)
	e.GET("/v1/finetunes/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Model   string `json:"model"`
	Dataset string `json:"dataset"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	model := strings.TrimSpace(req.Model)
	dataset := strings.TrimSpace(req.Dataset)
	if model == "" {
		return errors.InvalidArgument("model is required", nil)
	}
	if dataset == "" {
		return errors.InvalidArgument("dataset is required", nil)
	}
	now := time.Now().UTC()
	job := &Job{
		ID:        uuid.NewString(),
		Model:     model,
		Dataset:   dataset,
		Status:    "queued",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	// Simulate immediate start for local/stub behavior.
	job.Status = "running"
	job.UpdatedAt = time.Now().UTC()
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, job)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("finetune job not found", nil)
	}
	return c.JSON(http.StatusOK, job)
}
