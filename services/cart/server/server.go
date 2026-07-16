// Package server implements the cart service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/cart/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the cart service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"cart"`
	Port        string `env:"PORT" env-default:"8088"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the cart HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the cart HTTP server with an in-memory store.
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
	e.POST("/v1/carts", s.create)
	e.GET("/v1/carts/:id", s.get)
	e.POST("/v1/carts/:id/items", s.addItem)
	e.DELETE("/v1/carts/:id/items/:sku", s.removeItem)
	e.POST("/v1/carts/:id/checkout", s.checkout)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	UserID string `json:"user_id"`
}

type addItemRequest struct {
	SKU         string `json:"sku"`
	Qty         int64  `json:"qty"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type itemResponse struct {
	SKU         string `json:"sku"`
	Qty         int64  `json:"qty"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type cartResponse struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id,omitempty"`
	Items     []itemResponse `json:"items"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type checkoutResponse struct {
	CartID     string         `json:"cart_id"`
	UserID     string         `json:"user_id,omitempty"`
	Items      []itemResponse `json:"items"`
	Currency   string         `json:"currency"`
	TotalMinor int64          `json:"total_minor"`
	CheckedOut time.Time      `json:"checked_out"`
}

func toItems(items []store.Item) []itemResponse {
	out := make([]itemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, itemResponse{
			SKU:         it.SKU,
			Qty:         it.Qty,
			AmountMinor: it.AmountMinor,
			Currency:    it.Currency,
		})
	}
	return out
}

func toCartResponse(c *store.Cart) cartResponse {
	return cartResponse{
		ID:        c.ID,
		UserID:    c.UserID,
		Items:     toItems(c.Items),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	_ = c.Bind(&req) // body optional
	cart, err := s.store.Create(c.Request().Context(), strings.TrimSpace(req.UserID))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toCartResponse(cart))
}

func (s *Server) get(c echo.Context) error {
	cart, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toCartResponse(cart))
}

func (s *Server) addItem(c echo.Context) error {
	var req addItemRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	cart, err := s.store.AddItem(c.Request().Context(), c.Param("id"), store.AddItemInput{
		SKU:         strings.TrimSpace(req.SKU),
		Qty:         req.Qty,
		AmountMinor: req.AmountMinor,
		Currency:    strings.TrimSpace(req.Currency),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toCartResponse(cart))
}

func (s *Server) removeItem(c echo.Context) error {
	cart, err := s.store.RemoveItem(c.Request().Context(), c.Param("id"), c.Param("sku"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toCartResponse(cart))
}

func (s *Server) checkout(c echo.Context) error {
	result, err := s.store.Checkout(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, checkoutResponse{
		CartID:     result.CartID,
		UserID:     result.UserID,
		Items:      toItems(result.Items),
		Currency:   result.Currency,
		TotalMinor: result.TotalMinor,
		CheckedOut: result.CheckedOut,
	})
}
