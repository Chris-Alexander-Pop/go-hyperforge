// Package crudserver provides a reusable REST CRUD surface over memstore.
package crudserver

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
	"github.com/labstack/echo/v4"
)

// Config configures a CRUD HTTP server.
type Config struct {
	ServiceName string
	Port        string
	// Resource is the path segment under /v1/ (e.g. "products").
	Resource string
}

// Server is a REST CRUD server backed by memstore.
type Server struct {
	rest     *rest.Server
	store    *memstore.Store
	resource string
}

// New constructs a CRUD server.
func New(cfg Config) *Server {
	return NewWithStore(cfg, memstore.New())
}

// NewWithStore constructs a CRUD server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: st, resource: cfg.Resource}
	s.routes()
	return s
}

// Echo exposes the underlying Echo instance.
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) routes() {
	e := s.rest.Echo()
	base := "/v1/" + s.resource
	e.GET("/healthz", s.health)
	e.POST(base, s.create)
	e.GET(base, s.list)
	e.GET(base+"/:id", s.get)
	e.PUT(base+"/:id", s.update)
	e.DELETE(base+"/:id", s.remove)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) create(c echo.Context) error {
	var body map[string]interface{}
	if err := c.Bind(&body); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	rec, err := s.store.Create(c.Request().Context(), body)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, rec)
}

func (s *Server) list(c echo.Context) error {
	recs, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"items": recs})
}

func (s *Server) get(c echo.Context) error {
	rec, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rec)
}

func (s *Server) update(c echo.Context) error {
	var body map[string]interface{}
	if err := c.Bind(&body); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	rec, err := s.store.Update(c.Request().Context(), c.Param("id"), body)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rec)
}

func (s *Server) remove(c echo.Context) error {
	if err := s.store.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
