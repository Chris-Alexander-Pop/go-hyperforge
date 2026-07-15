package payment

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
)

// Config holds configuration for payment providers.
type Config struct {
	// Provider specifies the backend: "memory", "stripe", "paypal".
	Provider string `env:"PAYMENT_PROVIDER" env-default:"memory"`

	// Stripe configuration.
	StripeKey           string `env:"STRIPE_KEY"`
	StripeWebhookSecret string `env:"STRIPE_WEBHOOK_SECRET"`

	// PayPal configuration.
	PayPalClientID  string `env:"PAYPAL_CLIENT_ID"`
	PayPalSecret    string `env:"PAYPAL_SECRET"`
	PayPalSandbox   bool   `env:"PAYPAL_SANDBOX" env-default:"true"`
	PayPalWebhookID string `env:"PAYPAL_WEBHOOK_ID"`

	// Resilience (wired into Stripe/PayPal adapters).
	RetryMaxAttempts int           `env:"PAYMENT_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"PAYMENT_RETRY_BACKOFF" env-default:"100ms"`
}

// TransactionStatus represents the state of a payment.
type TransactionStatus string

const (
	StatusPending    TransactionStatus = "pending"
	StatusAuthorized TransactionStatus = "authorized"
	StatusSucceeded  TransactionStatus = "succeeded"
	StatusFailed     TransactionStatus = "failed"
	StatusRefunded   TransactionStatus = "refunded"
	StatusVoided     TransactionStatus = "voided"
)

// ChargeRequest represents a payment charge.
type ChargeRequest struct {
	// Amount is the charge in minor units (preferred over legacy float APIs).
	Amount commerce.Money

	// SourceID is a token or PaymentMethod ID.
	SourceID string

	// Description is an optional human-readable note.
	Description string

	// Metadata is opaque key/value data stored with the transaction.
	Metadata map[string]string

	// IdempotencyKey, when set, prevents duplicate charges for the same key.
	IdempotencyKey string
}

// RefundRequest represents a refund.
type RefundRequest struct {
	TransactionID string

	// Amount is optional; zero means full refund of the original transaction.
	Amount commerce.Money

	Reason string
}

// CaptureRequest represents a capture of a previously authorized payment.
type CaptureRequest struct {
	TransactionID string

	// Amount is optional; zero means capture the full authorized amount.
	Amount commerce.Money
}

// Transaction represents a payment transaction.
type Transaction struct {
	ID            string
	Amount        commerce.Money
	Status        TransactionStatus
	SourceID      string
	Description   string
	FailureReason string
	CreatedAt     time.Time
	Metadata      map[string]string

	// IdempotencyKey echoes the key used for Charge, when present.
	IdempotencyKey string
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

// Authorizer extends Provider with separate authorize / capture / void flows.
// Adapters that support auth-capture implement this interface; Charge remains
// a single-step authorize-and-capture for callers that do not need the split.
type Authorizer interface {
	Provider

	// Authorize places a hold without capturing funds.
	Authorize(ctx context.Context, req *ChargeRequest) (*Transaction, error)

	// Capture captures a previously authorized transaction.
	Capture(ctx context.Context, req *CaptureRequest) (*Transaction, error)

	// Void releases a previously authorized hold.
	Void(ctx context.Context, transactionID string) (*Transaction, error)
}

// WebhookEvent is a normalized payment provider webhook payload.
type WebhookEvent struct {
	ID        string
	Type      string
	Provider  string
	CreatedAt time.Time
	// Raw is the verified payload bytes (JSON).
	Raw []byte
	// Data holds provider-specific parsed fields when available.
	Data map[string]string
}

// WebhookVerifier verifies provider webhook signatures.
type WebhookVerifier interface {
	// Verify validates the signature over payload using headers and returns
	// a normalized event. Returns ErrInvalidWebhook on failure.
	Verify(ctx context.Context, payload []byte, headers map[string]string) (*WebhookEvent, error)
}
