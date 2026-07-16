// Package server implements the invoice service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/billing"
	billingmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/billing/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the invoice service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"invoice"`
	Port        string `env:"PORT" env-default:"8123"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the invoice HTTP API.
type Server struct {
	rest    *rest.Server
	billing billing.Service
	cfg     Config
}

// New constructs the invoice HTTP server with billing memory.
func New(cfg Config) *Server {
	return NewWithService(cfg, billingmemory.New())
}

// NewWithService constructs the server with a custom billing.Service (tests).
func NewWithService(cfg Config, svc billing.Service) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:    r,
		billing: svc,
		cfg:     cfg,
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
	e.POST("/v1/invoices", s.create)
	e.GET("/v1/invoices", s.list)
	e.GET("/v1/invoices/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	CustomerID  string `json:"customer_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type moneyJSON struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type invoiceResponse struct {
	ID             string     `json:"id"`
	SubscriptionID string     `json:"subscription_id,omitempty"`
	CustomerID     string     `json:"customer_id"`
	Amount         moneyJSON  `json:"amount"`
	Status         string     `json:"status"`
	IssuedAt       time.Time  `json:"issued_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	Description    string     `json:"description,omitempty"`
}

func moneyFrom(m commerce.Money) moneyJSON {
	return moneyJSON{AmountMinor: m.Amount, Currency: m.Currency}
}

func toInvoice(inv *billing.Invoice) invoiceResponse {
	return invoiceResponse{
		ID:             inv.ID,
		SubscriptionID: inv.SubscriptionID,
		CustomerID:     inv.CustomerID,
		Amount:         moneyFrom(inv.Amount),
		Status:         inv.Status,
		IssuedAt:       inv.IssuedAt,
		PaidAt:         inv.PaidAt,
		Description:    inv.Description,
	}
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return errors.InvalidArgument("customer_id is required", nil)
	}
	if strings.TrimSpace(req.Currency) == "" {
		return errors.InvalidArgument("currency is required", nil)
	}
	inv, err := s.billing.CreateInvoice(
		c.Request().Context(),
		strings.TrimSpace(req.CustomerID),
		commerce.NewMoney(req.AmountMinor, req.Currency),
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toInvoice(inv))
}

func (s *Server) list(c echo.Context) error {
	customerID := c.QueryParam("customer_id")
	invoices, err := s.billing.ListInvoices(c.Request().Context(), customerID)
	if err != nil {
		return err
	}
	out := make([]invoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, toInvoice(inv))
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) get(c echo.Context) error {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	// billing.Service has no GetInvoice; scan list.
	invoices, err := s.billing.ListInvoices(c.Request().Context(), "")
	if err != nil {
		return err
	}
	for _, inv := range invoices {
		if inv.ID == id {
			return c.JSON(http.StatusOK, toInvoice(inv))
		}
	}
	return billing.ErrInvoiceNotFound
}
