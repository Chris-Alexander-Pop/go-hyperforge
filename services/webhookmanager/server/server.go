// Package server implements the webhookmanager service HTTP API.
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

// Config is the webhookmanager service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"webhookmanager"`
	Port        string `env:"PORT" env-default:"8130"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Endpoint is a registered webhook destination.
type Endpoint struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Delivery is a recorded webhook delivery attempt.
type Delivery struct {
	ID         string    `json:"id"`
	EndpointID string    `json:"endpoint_id"`
	Event      string    `json:"event"`
	Payload    string    `json:"payload,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Server wraps the webhooks HTTP API.
type Server struct {
	rest       *rest.Server
	cfg        Config
	mu         sync.RWMutex
	endpoints  map[string]Endpoint
	deliveries []Delivery
}

// New constructs the webhookmanager HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, endpoints: make(map[string]Endpoint), deliveries: make([]Delivery, 0)}
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
	e.POST("/v1/webhooks", s.register)
	e.POST("/v1/webhooks/:id/deliveries", s.recordDelivery)
	e.GET("/v1/webhooks/:id/deliveries", s.listDeliveries)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events,omitempty"`
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		return errors.InvalidArgument("url is required", nil)
	}
	ep := Endpoint{ID: uuid.NewString(), URL: url, Events: req.Events, CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.endpoints[ep.ID] = ep
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, ep)
}

type deliveryRequest struct {
	Event   string `json:"event"`
	Payload string `json:"payload,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (s *Server) recordDelivery(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	var req deliveryRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	event := strings.TrimSpace(req.Event)
	if event == "" {
		return errors.InvalidArgument("event is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpoints[id]; !ok {
		return errors.NotFound("webhook endpoint not found", nil)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "delivered"
	}
	d := Delivery{
		ID:         uuid.NewString(),
		EndpointID: id,
		Event:      event,
		Payload:    req.Payload,
		Status:     status,
		CreatedAt:  time.Now().UTC(),
	}
	s.deliveries = append(s.deliveries, d)
	return c.JSON(http.StatusCreated, d)
}

func (s *Server) listDeliveries(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.endpoints[id]; !ok {
		return errors.NotFound("webhook endpoint not found", nil)
	}
	out := make([]Delivery, 0)
	for _, d := range s.deliveries {
		if d.EndpointID == id {
			out = append(out, d)
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"deliveries": out})
}
