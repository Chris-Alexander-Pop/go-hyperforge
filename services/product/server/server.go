// Package server implements the product service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/product/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the product service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"product"`
	Port        string `env:"PORT" env-default:"8087"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the product HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the product HTTP server with an in-memory store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, store.New())
}

// NewWithStore constructs the server with a custom store (tests).
func NewWithStore(cfg Config, st *store.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:  r,
		store: st,
		cfg:   cfg,
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
	e.POST("/v1/products", s.create)
	e.GET("/v1/products", s.list)
	e.GET("/v1/products/:id", s.get)
	e.PUT("/v1/products/:id", s.update)
	e.DELETE("/v1/products/:id", s.delete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type productRequest struct {
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	PriceMinor  int64  `json:"price_minor"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

type productResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SKU         string    `json:"sku"`
	PriceMinor  int64     `json:"price_minor"`
	Currency    string    `json:"currency"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toProductResponse(p *store.Product) productResponse {
	return productResponse{
		ID:          p.ID,
		Name:        p.Name,
		SKU:         p.SKU,
		PriceMinor:  p.PriceMinor,
		Currency:    p.Currency,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req productRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	p, err := s.store.Create(c.Request().Context(), store.CreateInput{
		Name:        strings.TrimSpace(req.Name),
		SKU:         strings.TrimSpace(req.SKU),
		PriceMinor:  req.PriceMinor,
		Currency:    strings.TrimSpace(req.Currency),
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toProductResponse(p))
}

func (s *Server) get(c echo.Context) error {
	p, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toProductResponse(p))
}

func (s *Server) list(c echo.Context) error {
	products, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]productResponse, 0, len(products))
	for _, p := range products {
		out = append(out, toProductResponse(p))
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) update(c echo.Context) error {
	var req productRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	p, err := s.store.Update(c.Request().Context(), c.Param("id"), store.UpdateInput{
		Name:        strings.TrimSpace(req.Name),
		SKU:         strings.TrimSpace(req.SKU),
		PriceMinor:  req.PriceMinor,
		Currency:    strings.TrimSpace(req.Currency),
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toProductResponse(p))
}

func (s *Server) delete(c echo.Context) error {
	if err := s.store.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
