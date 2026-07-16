// Package server implements the order service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/order/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the order service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"order"`
	Port        string `env:"PORT" env-default:"8089"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the order HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the order HTTP server with an in-memory store.
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
	e.POST("/v1/orders", s.create)
	e.GET("/v1/orders", s.list)
	e.GET("/v1/orders/:id", s.get)
	e.POST("/v1/orders/:id/cancel", s.cancel)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type itemRequest struct {
	SKU         string `json:"sku"`
	Qty         int64  `json:"qty"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type createRequest struct {
	CustomerID string        `json:"customer_id"`
	Items      []itemRequest `json:"items"`
	Currency   string        `json:"currency"`
}

type itemResponse struct {
	SKU         string `json:"sku"`
	Qty         int64  `json:"qty"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type orderResponse struct {
	ID         string         `json:"id"`
	CustomerID string         `json:"customer_id"`
	Items      []itemResponse `json:"items"`
	Currency   string         `json:"currency"`
	TotalMinor int64          `json:"total_minor"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func toOrderResponse(o *store.Order) orderResponse {
	items := make([]itemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, itemResponse{
			SKU:         it.SKU,
			Qty:         it.Qty,
			AmountMinor: it.AmountMinor,
			Currency:    it.Currency,
		})
	}
	return orderResponse{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		Items:      items,
		Currency:   o.Currency,
		TotalMinor: o.TotalMinor,
		Status:     string(o.Status),
		CreatedAt:  o.CreatedAt,
		UpdatedAt:  o.UpdatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	items := make([]store.Item, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, store.Item{
			SKU:         strings.TrimSpace(it.SKU),
			Qty:         it.Qty,
			AmountMinor: it.AmountMinor,
			Currency:    strings.TrimSpace(it.Currency),
		})
	}
	o, err := s.store.Create(c.Request().Context(), store.CreateInput{
		CustomerID: strings.TrimSpace(req.CustomerID),
		Items:      items,
		Currency:   strings.TrimSpace(req.Currency),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toOrderResponse(o))
}

func (s *Server) get(c echo.Context) error {
	o, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toOrderResponse(o))
}

func (s *Server) list(c echo.Context) error {
	orders, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, toOrderResponse(o))
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) cancel(c echo.Context) error {
	o, err := s.store.Cancel(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toOrderResponse(o))
}
