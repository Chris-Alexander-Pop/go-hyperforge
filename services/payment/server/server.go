// Package server implements the payment service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/payment"
	paymentmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/payment/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the payment service environment configuration.
type Config struct {
	ServiceName     string `env:"SERVICE_NAME" env-default:"payment"`
	Port            string `env:"PORT" env-default:"8090"`
	LogLevel        string `env:"LOG_LEVEL" env-default:"info"`
	PaymentProvider string `env:"PAYMENT_PROVIDER" env-default:"memory"`
}

// Server wraps the payment HTTP API.
type Server struct {
	rest     *rest.Server
	provider payment.Provider
	cfg      Config
}

// New constructs the payment HTTP server with the configured provider.
func New(cfg Config) *Server {
	provider := newProvider(cfg)
	return NewWithProvider(cfg, provider)
}

// NewWithProvider constructs the server with a custom payment.Provider (tests).
func NewWithProvider(cfg Config, provider payment.Provider) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:     r,
		provider: provider,
		cfg:      cfg,
	}
	s.routes()
	return s
}

func newProvider(cfg Config) payment.Provider {
	switch strings.ToLower(strings.TrimSpace(cfg.PaymentProvider)) {
	case "", "memory":
		return payment.NewInstrumentedProvider(paymentmemory.New())
	default:
		// Only memory is supported in this wave; fall back so local boot still works.
		return payment.NewInstrumentedProvider(paymentmemory.New())
	}
}

// Echo exposes the underlying Echo instance (tests / custom mounts).
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.provider != nil {
		_ = s.provider.Close()
	}
	return s.rest.Shutdown(ctx)
}

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/payments/charge", s.charge)
	e.POST("/v1/payments/refund", s.refund)
	e.GET("/v1/payments/:id", s.getTransaction)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type chargeRequest struct {
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	SourceID       string `json:"source_id"`
	Description    string `json:"description"`
	IdempotencyKey string `json:"idempotency_key"`
}

type refundRequest struct {
	TransactionID string `json:"transaction_id"`
	AmountMinor   *int64 `json:"amount_minor"`
	Currency      string `json:"currency"`
	Reason        string `json:"reason"`
}

type transactionResponse struct {
	ID             string    `json:"id"`
	AmountMinor    int64     `json:"amount_minor"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	SourceID       string    `json:"source_id"`
	Description    string    `json:"description,omitempty"`
	FailureReason  string    `json:"failure_reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
}

func toTransactionResponse(tx *payment.Transaction) transactionResponse {
	return transactionResponse{
		ID:             tx.ID,
		AmountMinor:    tx.Amount.Amount,
		Currency:       tx.Amount.Currency,
		Status:         string(tx.Status),
		SourceID:       tx.SourceID,
		Description:    tx.Description,
		FailureReason:  tx.FailureReason,
		CreatedAt:      tx.CreatedAt,
		IdempotencyKey: tx.IdempotencyKey,
	}
}

func (s *Server) charge(c echo.Context) error {
	var req chargeRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.Currency) == "" {
		return errors.InvalidArgument("currency is required", nil)
	}
	if strings.TrimSpace(req.SourceID) == "" {
		return errors.InvalidArgument("source_id is required", nil)
	}

	tx, err := s.provider.Charge(c.Request().Context(), &payment.ChargeRequest{
		Amount:         commerce.NewMoney(req.AmountMinor, req.Currency),
		SourceID:       req.SourceID,
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toTransactionResponse(tx))
}

func (s *Server) refund(c echo.Context) error {
	var req refundRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if strings.TrimSpace(req.TransactionID) == "" {
		return errors.InvalidArgument("transaction_id is required", nil)
	}

	refundReq := &payment.RefundRequest{
		TransactionID: req.TransactionID,
		Reason:        req.Reason,
	}
	if req.AmountMinor != nil {
		refundReq.Amount = commerce.NewMoney(*req.AmountMinor, req.Currency)
	}

	tx, err := s.provider.Refund(c.Request().Context(), refundReq)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toTransactionResponse(tx))
}

func (s *Server) getTransaction(c echo.Context) error {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	tx, err := s.provider.GetTransaction(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toTransactionResponse(tx))
}
