package billing

import (
	"context"
	"time"
)

// Config holds billing configuration.
type Config struct {
	// Provider: "memory", "stripe". Defaults to memory for local dev.
	Provider string `env:"BILLING_PROVIDER" env-default:"memory"`
}

// SubscriptionStatus represents subscription state.
type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusPastDue  SubscriptionStatus = "past_due"
)

// Subscription represents a recurring billing agreement.
type Subscription struct {
	ID         string
	CustomerID string
	PlanID     string
	Status     SubscriptionStatus
	Amount     float64
	Currency   string
	Interval   string // "month", "year"
	NextBillAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Invoice represents a bill for a specific period.
type Invoice struct {
	ID             string
	SubscriptionID string
	CustomerID     string
	Amount         float64
	Currency       string
	Status         string // "paid", "open", "void"
	IssuedAt       time.Time
	PaidAt         *time.Time
}

// Service defines the billing operations.
type Service interface {
	// CreateSubscription creates a new subscription.
	CreateSubscription(ctx context.Context, customerID string, planID string) (*Subscription, error)

	// CancelSubscription cancels an existing subscription.
	CancelSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)

	// GetSubscription retrieves a subscription.
	GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)

	// CreateInvoice creates a one-off invoice.
	CreateInvoice(ctx context.Context, customerID string, amount float64, currency string) (*Invoice, error)
}
