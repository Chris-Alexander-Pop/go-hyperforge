// Package server implements the taxcalculator service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/tax"
	taxmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/tax/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the taxcalculator service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"taxcalculator"`
	Port        string `env:"PORT" env-default:"8124"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the tax calculator HTTP API.
type Server struct {
	rest       *rest.Server
	calculator tax.Calculator
	cfg        Config
}

// New constructs the taxcalculator HTTP server with a memory adapter.
func New(cfg Config) *Server {
	return NewWithCalculator(cfg, taxmemory.New())
}

// NewWithCalculator constructs the server with a custom tax.Calculator (tests).
func NewWithCalculator(cfg Config, calc tax.Calculator) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:       r,
		calculator: calc,
		cfg:        cfg,
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
	e.POST("/v1/taxes/calculate", s.calculate)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type locationRequest struct {
	Country    string `json:"country"`
	State      string `json:"state"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
}

type calculateRequest struct {
	AmountMinor int64           `json:"amount_minor"`
	Currency    string          `json:"currency"`
	Location    locationRequest `json:"location"`
	// Flat fields accepted as an alternative to nested location.
	Country    string `json:"country"`
	State      string `json:"state"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
}

type moneyJSON struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type calculateResponse struct {
	TotalTax      moneyJSON            `json:"total_tax"`
	Rate          float64              `json:"rate"`
	Breakdown     map[string]moneyJSON `json:"breakdown"`
	TaxableAmount moneyJSON            `json:"taxable_amount"`
	Jurisdiction  struct {
		Country string `json:"country"`
		State   string `json:"state,omitempty"`
	} `json:"jurisdiction"`
}

func (s *Server) calculate(c echo.Context) error {
	var req calculateRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.Currency) == "" {
		return errors.InvalidArgument("currency is required", nil)
	}

	loc := tax.Location{
		Country:    firstNonEmpty(req.Location.Country, req.Country),
		State:      firstNonEmpty(req.Location.State, req.State),
		City:       firstNonEmpty(req.Location.City, req.City),
		PostalCode: firstNonEmpty(req.Location.PostalCode, req.PostalCode),
	}
	if strings.TrimSpace(loc.Country) == "" {
		return errors.InvalidArgument("location.country is required", nil)
	}

	result, err := s.calculator.CalculateTax(
		c.Request().Context(),
		commerce.NewMoney(req.AmountMinor, req.Currency),
		loc,
	)
	if err != nil {
		return err
	}

	breakdown := make(map[string]moneyJSON, len(result.Breakdown))
	for k, v := range result.Breakdown {
		breakdown[k] = moneyJSON{AmountMinor: v.Amount, Currency: v.Currency}
	}
	resp := calculateResponse{
		TotalTax:      moneyJSON{AmountMinor: result.TotalTax.Amount, Currency: result.TotalTax.Currency},
		Rate:          result.Rate,
		Breakdown:     breakdown,
		TaxableAmount: moneyJSON{AmountMinor: result.TaxableAmount.Amount, Currency: result.TaxableAmount.Currency},
	}
	resp.Jurisdiction.Country = result.Jurisdiction.Country
	resp.Jurisdiction.State = result.Jurisdiction.State
	return c.JSON(http.StatusOK, resp)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
