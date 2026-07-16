// Package server implements the permission service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rbac"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the permission service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"permission"`
	Port        string `env:"PORT" env-default:"8083"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

type permKey struct {
	Subject  string
	Resource string
	Action   string
}

// Server wraps the permissions HTTP API.
type Server struct {
	rest     *rest.Server
	cfg      Config
	mu       sync.RWMutex
	grants   map[permKey]struct{}
	enforcer rbac.Enforcer
}

// New constructs the permission HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:     r,
		cfg:      cfg,
		grants:   make(map[permKey]struct{}),
		enforcer: rbac.New(),
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
	e.POST("/v1/permissions/grant", s.grant)
	e.POST("/v1/permissions/revoke", s.revoke)
	e.POST("/v1/permissions/check", s.check)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type permRequest struct {
	Subject  string `json:"subject"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (s *Server) parse(c echo.Context) (permKey, error) {
	var req permRequest
	if err := c.Bind(&req); err != nil {
		return permKey{}, errors.InvalidArgument("invalid JSON body", err)
	}
	k := permKey{
		Subject:  strings.TrimSpace(req.Subject),
		Resource: strings.TrimSpace(req.Resource),
		Action:   strings.TrimSpace(req.Action),
	}
	if k.Subject == "" || k.Resource == "" || k.Action == "" {
		return permKey{}, errors.InvalidArgument("subject, resource, and action are required", nil)
	}
	return k, nil
}

func (s *Server) grant(c echo.Context) error {
	k, err := s.parse(c)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.grants[k] = struct{}{}
	s.mu.Unlock()
	s.enforcer.AddPolicy(k.Subject, k.Resource, k.Action)
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"subject":  k.Subject,
		"resource": k.Resource,
		"action":   k.Action,
		"granted":  true,
	})
}

func (s *Server) revoke(c echo.Context) error {
	k, err := s.parse(c)
	if err != nil {
		return err
	}
	s.mu.Lock()
	_, existed := s.grants[k]
	delete(s.grants, k)
	s.mu.Unlock()
	if !existed {
		return errors.NotFound("permission not found", nil)
	}
	// Rebuild enforcer without revoked grant (SimpleEnforcer has no Remove).
	s.rebuildEnforcer()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject":  k.Subject,
		"resource": k.Resource,
		"action":   k.Action,
		"revoked":  true,
	})
}

func (s *Server) rebuildEnforcer() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e := rbac.New()
	for k := range s.grants {
		e.AddPolicy(k.Subject, k.Resource, k.Action)
	}
	s.enforcer = e
}

func (s *Server) check(c echo.Context) error {
	k, err := s.parse(c)
	if err != nil {
		return err
	}
	s.mu.RLock()
	_, ok := s.grants[k]
	s.mu.RUnlock()
	allowed := ok
	if !allowed {
		// Also honor rbac admin shortcut via enforcer.
		allowed, err = s.enforcer.Enforce(c.Request().Context(), k.Subject, k.Resource, k.Action)
		if err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject":  k.Subject,
		"resource": k.Resource,
		"action":   k.Action,
		"allowed":  allowed,
	})
}
