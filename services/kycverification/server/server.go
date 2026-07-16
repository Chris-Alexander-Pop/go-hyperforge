// Package server implements the kycverification service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/kycverification/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the kycverification service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"kycverification"`
	Port        string `env:"PORT" env-default:"8132"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the KYC verification HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the kycverification HTTP server with an in-memory store.
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
	e.POST("/v1/kyc/applications", s.submit)
	e.GET("/v1/kyc/applications/:id", s.get)
	e.POST("/v1/kyc/applications/:id/approve", s.approve)
	e.POST("/v1/kyc/applications/:id/reject", s.reject)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type submitRequest struct {
	SubjectID string `json:"subject_id"`
	FullName  string `json:"full_name"`
	Document  string `json:"document"`
}

type applicationResponse struct {
	ID        string    `json:"id"`
	SubjectID string    `json:"subject_id"`
	FullName  string    `json:"full_name"`
	Document  string    `json:"document,omitempty"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(a *store.Application) applicationResponse {
	return applicationResponse{
		ID:        a.ID,
		SubjectID: a.SubjectID,
		FullName:  a.FullName,
		Document:  a.Document,
		Status:    string(a.Status),
		Reason:    a.Reason,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

func (s *Server) submit(c echo.Context) error {
	var req submitRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	app, err := s.store.Submit(c.Request().Context(), store.SubmitInput{
		SubjectID: strings.TrimSpace(req.SubjectID),
		FullName:  strings.TrimSpace(req.FullName),
		Document:  strings.TrimSpace(req.Document),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toResponse(app))
}

func (s *Server) get(c echo.Context) error {
	app, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(app))
}

func (s *Server) approve(c echo.Context) error {
	app, err := s.store.Approve(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(app))
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) reject(c echo.Context) error {
	var req rejectRequest
	_ = c.Bind(&req)
	app, err := s.store.Reject(c.Request().Context(), c.Param("id"), strings.TrimSpace(req.Reason))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toResponse(app))
}
