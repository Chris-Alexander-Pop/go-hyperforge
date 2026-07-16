// Package server implements the agentruntime service HTTP API.
package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/agents"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	llmmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the agentruntime service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"agentruntime"`
	Port        string `env:"PORT" env-default:"8096"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// RunStatus is the lifecycle status of an agent run.
type RunStatus string

const (
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

// Run is a persisted agent execution record.
type Run struct {
	ID        string    `json:"id"`
	Status    RunStatus `json:"status"`
	Goal      string    `json:"goal"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Server wraps the agentruntime HTTP API.
type Server struct {
	rest   *rest.Server
	client llm.Client
	cfg    Config

	mu   sync.RWMutex
	runs map[string]*Run
}

// New constructs the agentruntime HTTP server with an in-memory LLM client.
func New(cfg Config) *Server {
	// Memory LLM echoes by default; seed a catch-all FINAL ANSWER so ReAct completes.
	client := llmmemory.New().WithResponse("", "FINAL ANSWER: task completed")
	return NewWithClient(cfg, client)
}

// NewWithClient constructs the server with a custom llm.Client (tests).
func NewWithClient(cfg Config, client llm.Client) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:   r,
		client: client,
		cfg:    cfg,
		runs:   make(map[string]*Run),
	}
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
	e.POST("/v1/agents/runs", s.createRun)
	e.GET("/v1/agents/runs/:id", s.getRun)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type toolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createRunRequest struct {
	Goal  string     `json:"goal"`
	Tools []toolSpec `json:"tools,omitempty"`
}

func (s *Server) createRun(c echo.Context) error {
	var req createRunRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Goal == "" {
		return errors.InvalidArgument("goal is required", nil)
	}

	agentTools := make([]agents.Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		if t.Name == "" {
			continue
		}
		desc := t.Description
		if desc == "" {
			desc = t.Name
		}
		agentTools = append(agentTools, echoTool{name: t.Name, description: desc})
	}

	agent := agents.New(s.client, agentTools)
	output, err := agent.Run(c.Request().Context(), req.Goal)

	run := &Run{
		ID:        uuid.NewString(),
		Goal:      req.Goal,
		CreatedAt: time.Now().UTC(),
	}
	if err != nil {
		run.Status = RunStatusFailed
		run.Error = err.Error()
	} else {
		run.Status = RunStatusCompleted
		run.Output = output
	}

	s.mu.Lock()
	s.runs[run.ID] = run
	s.mu.Unlock()

	return c.JSON(http.StatusOK, run)
}

func (s *Server) getRun(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}

	s.mu.RLock()
	run, ok := s.runs[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("agent run not found", nil)
	}
	return c.JSON(http.StatusOK, run)
}

// echoTool is a minimal agents.Tool that echoes its input.
type echoTool struct {
	name        string
	description string
}

func (t echoTool) Name() string        { return t.name }
func (t echoTool) Description() string { return t.description }
func (t echoTool) Run(_ context.Context, input string) (string, error) {
	return input, nil
}
