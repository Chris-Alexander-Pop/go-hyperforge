// Package server implements the inventory service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/inventory/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the inventory service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"inventory"`
	Port        string `env:"PORT" env-default:"8091"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the inventory HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the inventory HTTP server with an in-memory store.
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
	e.POST("/v1/inventory/skus", s.upsert)
	e.GET("/v1/inventory/skus/:sku", s.get)
	e.POST("/v1/inventory/skus/:sku/reserve", s.reserve)
	e.POST("/v1/inventory/skus/:sku/release", s.release)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type upsertRequest struct {
	SKU      string `json:"sku"`
	Quantity int64  `json:"quantity"`
}

type qtyRequest struct {
	Qty int64 `json:"qty"`
}

type skuResponse struct {
	SKU       string    `json:"sku"`
	Quantity  int64     `json:"quantity"`
	Reserved  int64     `json:"reserved"`
	Available int64     `json:"available"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toSKUResponse(rec *store.SKU) skuResponse {
	return skuResponse{
		SKU:       rec.SKU,
		Quantity:  rec.Quantity,
		Reserved:  rec.Reserved,
		Available: rec.Available(),
		UpdatedAt: rec.UpdatedAt,
	}
}

func (s *Server) upsert(c echo.Context) error {
	var req upsertRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	rec, err := s.store.Upsert(c.Request().Context(), strings.TrimSpace(req.SKU), req.Quantity)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toSKUResponse(rec))
}

func (s *Server) get(c echo.Context) error {
	rec, err := s.store.Get(c.Request().Context(), c.Param("sku"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toSKUResponse(rec))
}

func (s *Server) reserve(c echo.Context) error {
	var req qtyRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	rec, err := s.store.Reserve(c.Request().Context(), c.Param("sku"), req.Qty)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toSKUResponse(rec))
}

func (s *Server) release(c echo.Context) error {
	var req qtyRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	rec, err := s.store.Release(c.Request().Context(), c.Param("sku"), req.Qty)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toSKUResponse(rec))
}
