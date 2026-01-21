package payment

import (
	"context"
	"time"
)

// Config holds configuration for payment providers.
type Config struct {
	// Provider specifies the backend: "memory", "stripe", "paypal".
	Provider string `env:"PAYMENT_PROVIDER" env-default:"memory"`

	// Stripe configuration.
	StripeKey string `env:"STRIPE_KEY"`

	// PayPal configuration.
	PayPalClientID string `env:"PAYPAL_CLIENT_ID"`
	PayPalSecret   string `env:"PAYPAL_SECRET"`
	PayPalSandbox  bool   `env:"PAYPAL_SANDBOX" env-default:"true"`
}

// TransactionStatus represents the state of a payment.
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusSucceeded TransactionStatus = "succeeded"
	StatusFailed    TransactionStatus = "failed"
	StatusRefunded  TransactionStatus = "refunded"
)

// ChargeRequest represents a payment charge.
type ChargeRequest struct {
	Amount      float64
	Currency    string
	SourceID    string // Token or PaymentMethod ID
	Description string
	Metadata    map[string]string
}

// RefundRequest represents a refund.
type RefundRequest struct {
	TransactionID string
	Amount        float64 // Optional, partial refund if non-zero
	Reason        string
}

// Transaction represents a payment transaction.
type Transaction struct {
	ID            string
	Amount        float64
	Currency      string
	Status        TransactionStatus
	SourceID      string
	Description   string
	FailureReason string
	CreatedAt     time.Time
	Metadata      map[string]string
}

// Provider defines the interface for payment processing.
type Provider interface {
	// Charge authorizes and captures a payment.
	Charge(ctx context.Context, req *ChargeRequest) (*Transaction, error)

	// Refund refunds a previous transaction.
	Refund(ctx context.Context, req *RefundRequest) (*Transaction, error)

	// GetTransaction retrieves a transaction by ID.
	GetTransaction(ctx context.Context, id string) (*Transaction, error)

	// Close releases resources.
	Close() error
}
