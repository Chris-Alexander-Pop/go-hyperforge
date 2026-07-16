// Package server implements the pricing service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/services/pricing/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the pricing service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"pricing"`
	Port        string `env:"PORT" env-default:"8112"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the pricing HTTP API.
type Server struct {
	rest  *rest.Server
	store *store.Store
	cfg   Config
}

// New constructs the pricing HTTP server with an in-memory store.
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
	e.POST("/v1/prices", s.create)
	e.GET("/v1/prices", s.list)
	e.GET("/v1/prices/:id", s.get)
	e.POST("/v1/prices/quote", s.quote)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type quoteRequest struct {
	SKU string `json:"sku"`
	Qty int64  `json:"qty"`
}

type priceResponse struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name,omitempty"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
}

type quoteResponse struct {
	SKU         string `json:"sku"`
	Qty         int64  `json:"qty"`
	Unit        int64  `json:"unit"`
	Total       int64  `json:"total"`
	Currency    string `json:"currency"`
	PriceRuleID string `json:"price_rule_id,omitempty"`
}

func toPriceResponse(r *store.PriceRule) priceResponse {
	return priceResponse{
		ID:          r.ID,
		SKU:         r.SKU,
		Name:        r.Name,
		AmountMinor: r.AmountMinor,
		Currency:    r.Currency,
		CreatedAt:   r.CreatedAt,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	rule, err := s.store.Create(c.Request().Context(), store.CreateInput{
		SKU:         strings.TrimSpace(req.SKU),
		Name:        strings.TrimSpace(req.Name),
		AmountMinor: req.AmountMinor,
		Currency:    strings.TrimSpace(req.Currency),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toPriceResponse(rule))
}

func (s *Server) get(c echo.Context) error {
	rule, err := s.store.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toPriceResponse(rule))
}

func (s *Server) list(c echo.Context) error {
	rules, err := s.store.List(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]priceResponse, 0, len(rules))
	for _, r := range rules {
		out = append(out, toPriceResponse(r))
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) quote(c echo.Context) error {
	var req quoteRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	q, err := s.store.Quote(c.Request().Context(), strings.TrimSpace(req.SKU), req.Qty)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, quoteResponse{
		SKU:         q.SKU,
		Qty:         q.Qty,
		Unit:        q.UnitMinor,
		Total:       q.TotalMinor,
		Currency:    q.Currency,
		PriceRuleID: q.PriceRuleID,
	})
}
