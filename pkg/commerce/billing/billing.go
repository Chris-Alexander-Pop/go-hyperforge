package billing

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
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

// Plan describes a billable product tier.
type Plan struct {
	ID       string
	Name     string
	Amount   commerce.Money
	Interval string // "month", "year"
	Metadata map[string]string
}

// Subscription represents a recurring billing agreement.
type Subscription struct {
	ID         string
	CustomerID string
	PlanID     string
	Status     SubscriptionStatus
	Amount     commerce.Money
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
	Amount         commerce.Money
	Status         string // "paid", "open", "void", "past_due"
	IssuedAt       time.Time
	PaidAt         *time.Time
}

// Catalog looks up plans. Memory adapters ship a built-in catalog.
type Catalog interface {
	GetPlan(ctx context.Context, planID string) (*Plan, error)
	ListPlans(ctx context.Context) ([]*Plan, error)
}

// Service defines the billing operations.
type Service interface {
	Catalog

	// CreateSubscription creates a new subscription for a known plan.
	CreateSubscription(ctx context.Context, customerID string, planID string) (*Subscription, error)

	// CancelSubscription cancels an existing subscription.
	CancelSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)

	// GetSubscription retrieves a subscription.
	GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)

	// UpgradeSubscription moves a subscription to a different plan.
	// Proration is a stub in the memory adapter (amount updates immediately).
	UpgradeSubscription(ctx context.Context, subscriptionID string, newPlanID string) (*Subscription, error)

	// MarkPastDue sets StatusPastDue after a failed invoice payment (dunning hook).
	MarkPastDue(ctx context.Context, subscriptionID string) (*Subscription, error)

	// CreateInvoice creates a one-off invoice.
	CreateInvoice(ctx context.Context, customerID string, amount commerce.Money) (*Invoice, error)
}
