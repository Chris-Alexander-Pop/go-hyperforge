// Package server implements the workflow service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	workflowmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/memory"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the workflow service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"workflow"`
	Port        string `env:"PORT" env-default:"8094"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the workflow HTTP API.
type Server struct {
	rest   *rest.Server
	engine workflow.WorkflowEngine
	cfg    Config
}

// New constructs the workflow HTTP server with an in-memory engine.
func New(cfg Config) *Server {
	return NewWithEngine(cfg, workflowmemory.New())
}

// NewWithEngine constructs the server with a custom WorkflowEngine (tests).
func NewWithEngine(cfg Config, engine workflow.WorkflowEngine) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, engine: engine, cfg: cfg}
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
	e.POST("/v1/workflows/definitions", s.registerDefinition)
	e.POST("/v1/workflows/start", s.start)
	e.GET("/v1/workflows/executions/:id", s.getExecution)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type definitionRequest struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Version        string           `json:"version"`
	StartAt        string           `json:"start_at"`
	TimeoutSeconds int              `json:"timeout_seconds"`
	States         []workflow.State `json:"states"`
}

func (s *Server) registerDefinition(c echo.Context) error {
	var req definitionRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Name == "" && req.ID == "" {
		return errors.InvalidArgument("id or name is required", nil)
	}

	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}
	def := workflow.WorkflowDefinition{
		ID:             id,
		Name:           req.Name,
		Version:        req.Version,
		StartAt:        req.StartAt,
		TimeoutSeconds: req.TimeoutSeconds,
		States:         req.States,
	}
	if err := s.engine.RegisterWorkflow(c.Request().Context(), def); err != nil {
		return err
	}

	registered, err := s.engine.GetWorkflow(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, registered)
}

type startRequest struct {
	WorkflowID     string      `json:"workflow_id"`
	ExecutionID    string      `json:"execution_id,omitempty"`
	Input          interface{} `json:"input,omitempty"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
}

func (s *Server) start(c echo.Context) error {
	var req startRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.WorkflowID == "" {
		return errors.InvalidArgument("workflow_id is required", nil)
	}

	exec, err := s.engine.Start(c.Request().Context(), workflow.StartOptions{
		WorkflowID:     req.WorkflowID,
		ExecutionID:    req.ExecutionID,
		Input:          req.Input,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, exec)
}

func (s *Server) getExecution(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return errors.InvalidArgument("execution id is required", nil)
	}
	exec, err := s.engine.GetExecution(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, exec)
}
