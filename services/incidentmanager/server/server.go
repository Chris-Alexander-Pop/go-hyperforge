// Package server implements the incidentmanager service HTTP API.
package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the incidentmanager service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"incidentmanager"`
	Port        string `env:"PORT" env-default:"8146"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Incident is an operational incident record.
type Incident struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Server wraps the incidentmanager HTTP API.
type Server struct {
	rest *rest.Server
	mu   sync.RWMutex
	byID map[string]Incident
}

// New constructs the incidentmanager HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, byID: make(map[string]Incident)}
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
	e.GET("/healthz", s.health)
	e.POST("/v1/incidents", s.create)
	e.GET("/v1/incidents", s.list)
	e.GET("/v1/incidents/:id", s.get)
	e.POST("/v1/incidents/:id/ack", s.ack)
	e.POST("/v1/incidents/:id/resolve", s.resolve)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createReq struct {
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

func (s *Server) create(c echo.Context) error {
	var req createReq
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Title == "" {
		return errors.InvalidArgument("title is required", nil)
	}
	sev := req.Severity
	if sev == "" {
		sev = "medium"
	}
	now := time.Now().UTC()
	inc := Incident{
		ID:          uuid.NewString(),
		Title:       req.Title,
		Severity:    sev,
		Status:      "open",
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.mu.Lock()
	s.byID[inc.ID] = inc
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, inc)
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Incident, 0, len(s.byID))
	for _, inc := range s.byID {
		out = append(out, inc)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"items": out})
}

func (s *Server) get(c echo.Context) error {
	s.mu.RLock()
	inc, ok := s.byID[c.Param("id")]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("incident not found", nil)
	}
	return c.JSON(http.StatusOK, inc)
}

func (s *Server) ack(c echo.Context) error {
	return s.setStatus(c, "acknowledged")
}

func (s *Server) resolve(c echo.Context) error {
	return s.setStatus(c, "resolved")
}

func (s *Server) setStatus(c echo.Context, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	inc, ok := s.byID[c.Param("id")]
	if !ok {
		return errors.NotFound("incident not found", nil)
	}
	inc.Status = status
	inc.UpdatedAt = time.Now().UTC()
	s.byID[inc.ID] = inc
	return c.JSON(http.StatusOK, inc)
}
