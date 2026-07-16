// Package server implements the subscription service HTTP API.
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

// Config is the subscription service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"subscription"`
	Port        string `env:"PORT" env-default:"8125"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the subscription HTTP API.
type Server struct {
	rest    *rest.Server
	billing billing.Service
	cfg     Config
}

// New constructs the subscription HTTP server with billing memory.
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
	e.POST("/v1/subscriptions", s.create)
	e.GET("/v1/subscriptions/:id", s.get)
	e.POST("/v1/subscriptions/:id/cancel", s.cancel)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	CustomerID string `json:"customer_id"`
	PlanID     string `json:"plan_id"`
}

type moneyJSON struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
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

func moneyFrom(m commerce.Money) moneyJSON {
	return moneyJSON{AmountMinor: m.Amount, Currency: m.Currency}
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

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return errors.InvalidArgument("customer_id is required", nil)
	}
	if strings.TrimSpace(req.PlanID) == "" {
		return errors.InvalidArgument("plan_id is required", nil)
	}
	sub, err := s.billing.CreateSubscription(c.Request().Context(), strings.TrimSpace(req.CustomerID), strings.TrimSpace(req.PlanID))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toSubscription(sub))
}

func (s *Server) get(c echo.Context) error {
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

func (s *Server) cancel(c echo.Context) error {
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
