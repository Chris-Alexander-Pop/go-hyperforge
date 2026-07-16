// Package server implements the etlpipeline service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata/pipeline/etl"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the etlpipeline service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"etlpipeline"`
	Port        string `env:"PORT" env-default:"8139"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Job is an ETL job definition and status.
type Job struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	RowsIn    int       `json:"rows_in,omitempty"`
	RowsOut   int       `json:"rows_out,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type memoryExtractor struct{ rows []interface{} }
type memoryTransformer struct{}
type memoryLoader struct {
	out *[]interface{}
}

func (e *memoryExtractor) Extract(ctx context.Context, out chan<- interface{}) error {
	for _, r := range e.rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- r:
		}
	}
	return nil
}

func (t *memoryTransformer) Transform(ctx context.Context, in <-chan interface{}, out chan<- interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v, ok := <-in:
			if !ok {
				return nil
			}
			out <- v
		}
	}
}

func (l *memoryLoader) Load(ctx context.Context, in <-chan interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v, ok := <-in:
			if !ok {
				return nil
			}
			*l.out = append(*l.out, v)
		}
	}
}

// Server wraps the ETL HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	jobs map[string]*Job
}

// New constructs the etlpipeline HTTP server.
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
	e.POST("/v1/etl", s.create)
	e.POST("/v1/etl/:id/run", s.run)
	e.GET("/v1/etl/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Name string `json:"name"`
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
	now := time.Now().UTC()
	job := &Job{ID: uuid.NewString(), Name: name, Status: "created", CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, job)
}

type runRequest struct {
	Rows []interface{} `json:"rows,omitempty"`
}

func (s *Server) run(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	var req runRequest
	_ = c.Bind(&req)
	s.mu.Lock()
	job, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return errors.NotFound("etl job not found", nil)
	}
	job.Status = "running"
	job.UpdatedAt = time.Now().UTC()
	s.mu.Unlock()

	rows := req.Rows
	if rows == nil {
		rows = []interface{}{"a", "b", "c"}
	}
	loaded := make([]interface{}, 0)
	p := &etl.SimplePipeline{
		E: &memoryExtractor{rows: rows},
		T: &memoryTransformer{},
		L: &memoryLoader{out: &loaded},
	}
	err := p.Run(c.Request().Context())

	s.mu.Lock()
	job = s.jobs[id]
	job.RowsIn = len(rows)
	job.RowsOut = len(loaded)
	job.UpdatedAt = time.Now().UTC()
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
	} else {
		job.Status = "completed"
		job.Error = ""
	}
	out := *job
	s.mu.Unlock()
	if err != nil {
		return c.JSON(http.StatusOK, out)
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("etl job not found", nil)
	}
	return c.JSON(http.StatusOK, job)
}
