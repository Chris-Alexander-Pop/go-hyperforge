// Package server implements the identityprovider service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/identityprovider/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the identityprovider service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"identityprovider"`
	Port        string `env:"PORT" env-default:"8127"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the identity provider HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the identityprovider HTTP server with an in-memory store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, store.New())
}

// NewWithStore constructs the server with a custom store (tests).
func NewWithStore(cfg Config, st *store.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: st, cfg: cfg}
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
	e.POST("/v1/identities", s.create)
	e.GET("/v1/identities", s.list)
	e.GET("/v1/identities/:id", s.get)
	e.PUT("/v1/identities/:id", s.update)
	e.DELETE("/v1/identities/:id", s.delete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Password string   `json:"password"`
}

type updateRequest struct {
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Password string   `json:"password"`
}

type identityResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(id *store.Identity) identityResponse {
	roles := id.Roles
	if roles == nil {
		roles = []string{}
	}
	return identityResponse{
		ID:        id.ID,
		Username:  id.Username,
		Email:     id.Email,
		Roles:     roles,
		CreatedAt: id.CreatedAt,
		UpdatedAt: id.UpdatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	id, err := s.store.Create(c.Request().Context(), store.CreateInput{
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		Roles:    req.Roles,
		Password: req.Password,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(id))
}

func (s *Server) list(c echo.Context) error {
	items, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]identityResponse, 0, len(items))
	for _, id := range items {
		out = append(out, toResponse(id))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"identities": out})
}

func (s *Server) get(c echo.Context) error {
	id, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(id))
}

func (s *Server) update(c echo.Context) error {
	var req updateRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	id, err := s.store.Update(c.Request().Context(), c.Param("id"), store.UpdateInput{
		Email:    strings.TrimSpace(req.Email),
		Roles:    req.Roles,
		Password: req.Password,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(id))
}

func (s *Server) delete(c echo.Context) error {
	if err := s.store.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
