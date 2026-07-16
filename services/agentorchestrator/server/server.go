// Package server implements the agentorchestrator service HTTP API.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	llmmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/adapters/memory"
	convmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the agentorchestrator service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"agentorchestrator"`
	Port        string `env:"PORT" env-default:"8119"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// OrchestrationStatus is the lifecycle status of an orchestration.
type OrchestrationStatus string

const (
	OrchestrationCompleted OrchestrationStatus = "completed"
	OrchestrationFailed    OrchestrationStatus = "failed"
)

// Step is one planned orchestration step.
type Step struct {
	Index   int    `json:"index"`
	AgentID string `json:"agent_id,omitempty"`
	Action  string `json:"action"`
	Result  string `json:"result,omitempty"`
}

// Orchestration is a multi-step plan execution record.
type Orchestration struct {
	ID        string              `json:"id"`
	Status    OrchestrationStatus `json:"status"`
	Goal      string              `json:"goal"`
	AgentIDs  []string            `json:"agent_ids,omitempty"`
	Steps     []Step              `json:"steps,omitempty"`
	Output    string              `json:"output,omitempty"`
	Error     string              `json:"error,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
}

// Server wraps the agentorchestrator HTTP API.
type Server struct {
	rest   *rest.Server
	client llm.Client
	cfg    Config

	mu             sync.RWMutex
	orchestrations map[string]*Orchestration
}

// New constructs the agentorchestrator HTTP server with an in-memory LLM client.
func New(cfg Config) *Server {
	return NewWithClient(cfg, llmmemory.New())
}

// NewWithClient constructs the server with a custom llm.Client (tests).
func NewWithClient(cfg Config, client llm.Client) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:           r,
		client:         client,
		cfg:            cfg,
		orchestrations: make(map[string]*Orchestration),
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
	e.POST("/v1/orchestrations", s.create)
	e.GET("/v1/orchestrations/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Goal     string   `json:"goal"`
	AgentIDs []string `json:"agent_ids,omitempty"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Goal == "" {
		return errors.InvalidArgument("goal is required", nil)
	}

	orch := &Orchestration{
		ID:        uuid.NewString(),
		Goal:      req.Goal,
		AgentIDs:  append([]string(nil), req.AgentIDs...),
		CreatedAt: time.Now().UTC(),
	}

	steps, output, err := s.runPlan(c.Request().Context(), req.Goal, req.AgentIDs)
	orch.Steps = steps
	if err != nil {
		orch.Status = OrchestrationFailed
		orch.Error = err.Error()
	} else {
		orch.Status = OrchestrationCompleted
		orch.Output = output
	}

	s.mu.Lock()
	s.orchestrations[orch.ID] = orch
	s.mu.Unlock()

	return c.JSON(http.StatusOK, orch)
}

func (s *Server) get(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}

	s.mu.RLock()
	orch, ok := s.orchestrations[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("orchestration not found", nil)
	}
	return c.JSON(http.StatusOK, orch)
}

func (s *Server) runPlan(ctx context.Context, goal string, agentIDs []string) ([]Step, string, error) {
	mem := convmemory.NewSimpleMemory(50)
	if err := mem.AddUserMessage(ctx, fmt.Sprintf("Create a short multi-step plan for: %s", goal)); err != nil {
		return nil, "", err
	}

	agents := agentIDs
	if len(agents) == 0 {
		agents = []string{"planner", "executor"}
	}

	steps := make([]Step, 0, len(agents))
	for i, agentID := range agents {
		msgs, err := mem.GetMessages(ctx)
		if err != nil {
			return steps, "", err
		}
		prompt := append(append([]llm.Message{}, msgs...), llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Step %d for agent %s toward goal: %s", i+1, agentID, goal),
		})
		gen, err := s.client.Chat(ctx, prompt)
		if err != nil {
			return steps, "", err
		}
		result := gen.Message.Content
		if err := mem.AddAssistantMessage(ctx, result); err != nil {
			return steps, "", err
		}
		steps = append(steps, Step{
			Index:   i + 1,
			AgentID: agentID,
			Action:  fmt.Sprintf("execute step %d", i+1),
			Result:  result,
		})
	}

	finalMsgs, err := mem.GetMessages(ctx)
	if err != nil {
		return steps, "", err
	}
	summaryPrompt := append(append([]llm.Message{}, finalMsgs...), llm.Message{
		Role:    llm.RoleUser,
		Content: "Summarize the orchestration outcome for the goal.",
	})
	gen, err := s.client.Chat(ctx, summaryPrompt)
	if err != nil {
		return steps, "", err
	}
	return steps, gen.Message.Content, nil
}
