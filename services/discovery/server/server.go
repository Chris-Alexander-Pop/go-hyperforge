// Package server implements the discovery service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	discoverymemory "github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery/adapters/memory"
	"github.com/labstack/echo/v4"
)

// Config is the discovery service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"discovery"`
	Port        string `env:"PORT" env-default:"8106"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the service discovery HTTP API.
type Server struct {
	rest     *rest.Server
	registry discovery.ServiceRegistry
	cfg      Config
}

// New constructs the discovery HTTP server with an in-memory registry.
func New(cfg Config) *Server {
	return NewWithRegistry(cfg, discoverymemory.New())
}

// NewWithRegistry constructs the server with a custom registry (tests).
func NewWithRegistry(cfg Config, registry discovery.ServiceRegistry) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, registry: registry, cfg: cfg}
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
	e.POST("/v1/services", s.register)
	e.POST("/v1/services/:id/heartbeat", s.heartbeat)
	e.GET("/v1/services", s.listHealthy)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type serviceResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Address       string            `json:"address"`
	Port          int               `json:"port"`
	Tags          []string          `json:"tags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Health        string            `json:"health"`
	Namespace     string            `json:"namespace,omitempty"`
	Weight        int               `json:"weight,omitempty"`
	RegisteredAt  interface{}       `json:"registered_at,omitempty"`
	LastHeartbeat interface{}       `json:"last_heartbeat,omitempty"`
}

func toResponse(svc *discovery.Service) serviceResponse {
	if svc == nil {
		return serviceResponse{}
	}
	return serviceResponse{
		ID:            svc.ID,
		Name:          svc.Name,
		Address:       svc.Address,
		Port:          svc.Port,
		Tags:          svc.Tags,
		Metadata:      svc.Metadata,
		Health:        string(svc.Health),
		Namespace:     svc.Namespace,
		Weight:        svc.Weight,
		RegisteredAt:  svc.RegisteredAt,
		LastHeartbeat: svc.LastHeartbeat,
	}
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	svc, err := s.registry.Register(c.Request().Context(), discovery.RegisterOptions{
		Name:     name,
		Address:  strings.TrimSpace(req.Address),
		Port:     req.Port,
		Tags:     req.Tags,
		Metadata: req.Metadata,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(svc))
}

func (s *Server) heartbeat(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	if err := s.registry.Heartbeat(c.Request().Context(), id); err != nil {
		return err
	}
	svc, err := s.registry.Get(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(svc))
}

func (s *Server) listHealthy(c echo.Context) error {
	name := strings.TrimSpace(c.QueryParam("name"))
	opts := discovery.QueryOptions{HealthyOnly: true}
	var (
		svcs []*discovery.Service
		err  error
	)
	if name != "" {
		svcs, err = s.registry.Lookup(c.Request().Context(), name, opts)
	} else {
		svcs, err = s.registry.List(c.Request().Context(), opts)
	}
	if err != nil {
		return err
	}
	out := make([]serviceResponse, 0, len(svcs))
	for _, svc := range svcs {
		out = append(out, toResponse(svc))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"services": out})
}
