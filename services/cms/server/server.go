// Package server implements the cms service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the cms service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"cms"`
	Port        string `env:"PORT" env-default:"8117"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Page is a CMS page.
type Page struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// Server wraps the pages HTTP API.
type Server struct {
	rest   *rest.Server
	cfg    Config
	mu     sync.RWMutex
	byID   map[string]*Page
	bySlug map[string]string
}

// New constructs the cms HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, byID: make(map[string]*Page), bySlug: make(map[string]string)}
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
	e.POST("/v1/pages", s.create)
	e.POST("/v1/pages/:id/publish", s.publish)
	e.GET("/v1/pages/by-slug/:slug", s.getBySlug)
	e.GET("/v1/pages/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	slug := strings.TrimSpace(req.Slug)
	title := strings.TrimSpace(req.Title)
	if slug == "" {
		return errors.InvalidArgument("slug is required", nil)
	}
	if title == "" {
		return errors.InvalidArgument("title is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bySlug[slug]; exists {
		return errors.Conflict("slug already exists", nil)
	}
	p := &Page{
		ID:        uuid.NewString(),
		Slug:      slug,
		Title:     title,
		Body:      req.Body,
		Status:    "draft",
		CreatedAt: time.Now().UTC(),
	}
	s.byID[p.ID] = p
	s.bySlug[slug] = p.ID
	return c.JSON(http.StatusCreated, p)
}

func (s *Server) publish(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return errors.NotFound("page not found", nil)
	}
	p.Status = "published"
	p.PublishedAt = time.Now().UTC()
	return c.JSON(http.StatusOK, p)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	p, ok := s.byID[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("page not found", nil)
	}
	return c.JSON(http.StatusOK, p)
}

func (s *Server) getBySlug(c echo.Context) error {
	slug := strings.TrimSpace(c.Param("slug"))
	s.mu.RLock()
	id, ok := s.bySlug[slug]
	if !ok {
		s.mu.RUnlock()
		return errors.NotFound("page not found", nil)
	}
	p := s.byID[id]
	s.mu.RUnlock()
	return c.JSON(http.StatusOK, p)
}
