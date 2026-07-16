// Package server implements the billing service HTTP API.
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

// Config is the billing service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"billing"`
	Port        string `env:"PORT" env-default:"8122"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the billing HTTP API.
type Server struct {
	rest    *rest.Server
	billing billing.Service
	cfg     Config
}

// New constructs the billing HTTP server with a memory adapter.
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
	e.GET("/v1/bills/plans", s.listPlans)
	e.POST("/v1/bills/subscriptions", s.createSubscription)
	e.GET("/v1/bills/subscriptions/:id", s.getSubscription)
	e.POST("/v1/bills/subscriptions/:id/cancel", s.cancelSubscription)
	e.POST("/v1/bills/invoices", s.createInvoice)
	e.GET("/v1/bills/invoices", s.listInvoices)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type moneyJSON struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type planResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Amount   moneyJSON         `json:"amount"`
	Interval string            `json:"interval"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type subscriptionRequest struct {
	CustomerID string `json:"customer_id"`
	PlanID     string `json:"plan_id"`
}

type subscriptionResponse struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	PlanID     string    `json:"plan_id"`
	Status     string    `json:"status"`
	Amount     moneyJSON `json:"amount"`
	Interval   string    `json:"interval"`
	NextBillAt time.Time `json:"next_bill_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type invoiceRequest struct {
	CustomerID  string `json:"customer_id"`
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

func toPlan(p *billing.Plan) planResponse {
	return planResponse{
		ID:       p.ID,
		Name:     p.Name,
		Amount:   moneyFrom(p.Amount),
		Interval: p.Interval,
		Metadata: p.Metadata,
	}
}

func toSubscription(sub *billing.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:         sub.ID,
		CustomerID: sub.CustomerID,
		PlanID:     sub.PlanID,
		Status:     string(sub.Status),
		Amount:     moneyFrom(sub.Amount),
		Interval:   sub.Interval,
		NextBillAt: sub.NextBillAt,
		CreatedAt:  sub.CreatedAt,
		UpdatedAt:  sub.UpdatedAt,
	}
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

func (s *Server) listPlans(c echo.Context) error {
	plans, err := s.billing.ListPlans(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, toPlan(p))
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) createSubscription(c echo.Context) error {
	var req subscriptionRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return errors.InvalidArgument("customer_id is required", nil)
	}
	if strings.TrimSpace(req.PlanID) == "" {
		return errors.InvalidArgument("plan_id is required", nil)
	}
	sub, err := s.billing.CreateSubscription(c.Request().Context(), req.CustomerID, req.PlanID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toSubscription(sub))
}

func (s *Server) getSubscription(c echo.Context) error {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	sub, err := s.billing.GetSubscription(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toSubscription(sub))
}

func (s *Server) cancelSubscription(c echo.Context) error {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	sub, err := s.billing.CancelSubscription(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toSubscription(sub))
}

func (s *Server) createInvoice(c echo.Context) error {
	var req invoiceRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return errors.InvalidArgument("customer_id is required", nil)
	}
	if strings.TrimSpace(req.Currency) == "" {
		return errors.InvalidArgument("currency is required", nil)
	}
	inv, err := s.billing.CreateInvoice(c.Request().Context(), req.CustomerID, commerce.NewMoney(req.AmountMinor, req.Currency))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toInvoice(inv))
}

func (s *Server) listInvoices(c echo.Context) error {
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
