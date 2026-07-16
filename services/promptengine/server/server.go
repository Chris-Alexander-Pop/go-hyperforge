// Package server implements the promptengine service HTTP API.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt"
	promptmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the promptengine service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"promptengine"`
	Port        string `env:"PORT" env-default:"8101"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the promptengine HTTP API.
type Server struct {
	rest  *rest.Server
	store prompt.Store
	cfg   Config
}

// New constructs the promptengine HTTP server with an in-memory prompt store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, promptmemory.New())
}

// NewWithStore constructs the server with a custom prompt.Store (tests).
func NewWithStore(cfg Config, store prompt.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: store, cfg: cfg}
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
	e.POST("/v1/prompts", s.put)
	e.GET("/v1/prompts/:name", s.get)
	e.POST("/v1/prompts/:name/render", s.render)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type putRequest struct {
	Name     string `json:"name"`
	Template string `json:"template"`
	Version  string `json:"version,omitempty"`
}

type promptView struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Template string `json:"template"`
}

func (s *Server) put(c echo.Context) error {
	var req putRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	if req.Template == "" {
		return errors.InvalidArgument("template is required", nil)
	}
	version := req.Version
	if version == "" {
		version = "v1"
	}

	t := prompt.Template{Name: req.Name, Version: version, Body: req.Template}
	if err := s.store.Put(c.Request().Context(), t); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, promptView{
		Name:     t.Name,
		Version:  t.Version,
		Template: t.Body,
	})
}

func (s *Server) get(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	t, err := s.store.Get(c.Request().Context(), name, "")
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, promptView{
		Name:     t.Name,
		Version:  t.Version,
		Template: t.Body,
	})
}

type renderRequest struct {
	Vars    map[string]interface{} `json:"vars"`
	Version string                 `json:"version,omitempty"`
}

type renderResponse struct {
	Name     string `json:"name"`
	Rendered string `json:"rendered"`
}

func (s *Server) render(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	var req renderRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}

	vars := make(map[string]string, len(req.Vars))
	for k, v := range req.Vars {
		vars[k] = fmt.Sprint(v)
	}

	rendered, err := s.store.Render(c.Request().Context(), name, req.Version, vars)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, renderResponse{Name: name, Rendered: rendered})
}
