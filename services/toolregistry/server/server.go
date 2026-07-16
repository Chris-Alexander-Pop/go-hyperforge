// Package server implements the toolregistry service HTTP API.
package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/tools"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the toolregistry service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"toolregistry"`
	Port        string `env:"PORT" env-default:"8097"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// ToolView is the HTTP representation of a registered tool.
type ToolView struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// Server wraps the toolregistry HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config

	mu    sync.RWMutex
	tools map[string]tools.RegisteredTool
}

// New constructs the toolregistry HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:  r,
		cfg:   cfg,
		tools: make(map[string]tools.RegisteredTool),
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

// Registry returns a tools.Registry snapshot of currently registered tools.
func (s *Server) Registry() *tools.Registry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reg := tools.NewRegistry()
	for name, t := range s.tools {
		reg.Register(name, t.Def.Function.Description, t.Def.Function.Parameters, t.Func)
	}
	return reg
}

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/tools", s.register)
	e.GET("/v1/tools", s.list)
	e.GET("/v1/tools/:name", s.get)
	e.DELETE("/v1/tools/:name", s.delete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Name == "" {
		return errors.InvalidArgument("name is required", nil)
	}

	params := req.Parameters
	if params == nil {
		params = map[string]interface{}{}
	}

	fn := func(_ context.Context, _ []byte) (string, error) {
		return "", nil
	}

	s.mu.Lock()
	s.tools[req.Name] = tools.RegisteredTool{
		Def: llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        req.Name,
				Description: req.Description,
				Parameters:  params,
			},
		},
		Func: fn,
	}
	s.mu.Unlock()

	return c.JSON(http.StatusCreated, ToolView{
		Name:        req.Name,
		Description: req.Description,
		Parameters:  params,
	})
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	out := make([]ToolView, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, ToolView{
			Name:        t.Def.Function.Name,
			Description: t.Def.Function.Description,
			Parameters:  t.Def.Function.Parameters,
		})
	}
	s.mu.RUnlock()
	return c.JSON(http.StatusOK, out)
}

func (s *Server) get(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}

	s.mu.RLock()
	t, ok := s.tools[name]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("tool not found", nil)
	}
	return c.JSON(http.StatusOK, ToolView{
		Name:        t.Def.Function.Name,
		Description: t.Def.Function.Description,
		Parameters:  t.Def.Function.Parameters,
	})
}

func (s *Server) delete(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}

	s.mu.Lock()
	_, ok := s.tools[name]
	if ok {
		delete(s.tools, name)
	}
	s.mu.Unlock()
	if !ok {
		return errors.NotFound("tool not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}
