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

// Invoice status values.
const (
	InvoiceOpen    = "open"
	InvoicePaid    = "paid"
	InvoiceVoid    = "void"
	InvoicePastDue = "past_due"
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
	// PeriodStart is the start of the current billing period (for proration).
	PeriodStart time.Time
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
	// Description is an optional note (e.g. "proration").
	Description string
}

// Catalog looks up plans. Memory adapters ship a built-in catalog.
type Catalog interface {
	GetPlan(ctx context.Context, planID string) (*Plan, error)
	ListPlans(ctx context.Context) ([]*Plan, error)
}

// ProrationResult holds mid-cycle plan-change amounts.
type ProrationResult struct {
	// Credit is the unused portion of the old plan (positive money).
	Credit commerce.Money
	// Charge is the remaining portion of the new plan.
	Charge commerce.Money
	// Net is Charge − Credit. Positive means the customer owes; negative is credit.
	Net commerce.Money
	// Fraction is the remaining period fraction in [0, 1].
	Fraction float64
}

// DunningResult is returned by ProcessDunning.
type DunningResult struct {
	Subscription *Subscription
	Invoices     []*Invoice // invoices transitioned to past_due
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

	// UpgradeSubscription moves a subscription to a different plan, applying
	// proration for the remaining period and issuing a net proration invoice
	// when the customer owes a positive amount.
	UpgradeSubscription(ctx context.Context, subscriptionID string, newPlanID string) (*Subscription, error)

	// MarkPastDue sets StatusPastDue after a failed invoice payment (dunning hook).
	MarkPastDue(ctx context.Context, subscriptionID string) (*Subscription, error)

	// ProcessDunning transitions open invoices for the subscription to past_due
	// and marks the subscription StatusPastDue.
	ProcessDunning(ctx context.Context, subscriptionID string) (*DunningResult, error)

	// CreateInvoice creates a one-off invoice.
	CreateInvoice(ctx context.Context, customerID string, amount commerce.Money) (*Invoice, error)

	// ListInvoices returns invoices for a customer (optional subscription filter via empty = all).
	ListInvoices(ctx context.Context, customerID string) ([]*Invoice, error)
}

// Prorate computes mid-cycle plan-change credit/charge for the remaining period.
//
//	fraction = remaining / period
//	credit   = oldAmount × fraction
//	charge   = newAmount × fraction
//	net      = charge − credit
//
// Currencies on oldAmount and newAmount must match. changeAt outside
// [periodStart, periodEnd] is clamped to the period bounds.
func Prorate(oldAmount, newAmount commerce.Money, periodStart, periodEnd, changeAt time.Time) (*ProrationResult, error) {
	if err := oldAmount.Validate(); err != nil {
		return nil, err
	}
	if err := newAmount.Validate(); err != nil {
		return nil, err
	}
	if !oldAmount.SameCurrency(newAmount) {
		return nil, ErrCurrencyMismatch
	}
	if !periodEnd.After(periodStart) {
		return nil, ErrInvalidPeriod
	}

	if changeAt.Before(periodStart) {
		changeAt = periodStart
	}
	if changeAt.After(periodEnd) {
		changeAt = periodEnd
	}

	total := periodEnd.Sub(periodStart).Seconds()
	remaining := periodEnd.Sub(changeAt).Seconds()
	fraction := remaining / total
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	creditMinor := roundHalfAway(float64(oldAmount.Amount) * fraction)
	chargeMinor := roundHalfAway(float64(newAmount.Amount) * fraction)
	cur := oldAmount.Currency

	credit := commerce.NewMoney(creditMinor, cur)
	charge := commerce.NewMoney(chargeMinor, cur)
	net, err := charge.Sub(credit)
	if err != nil {
		return nil, err
	}

	return &ProrationResult{
		Credit:   credit,
		Charge:   charge,
		Net:      net,
		Fraction: fraction,
	}, nil
}

func roundHalfAway(v float64) int64 {
	if v < 0 {
		return int64(v - 0.5)
	}
	return int64(v + 0.5)
}
