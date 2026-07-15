package billing

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedService implements Service at compile time.
var _ Service = (*EventedService)(nil)

const (
	// TopicBilling is the pkg/events topic for billing domain events.
	TopicBilling = "commerce.billing"

	// EventTypeSubscriptionCreated is emitted after CreateSubscription.
	EventTypeSubscriptionCreated = "billing.subscription.created"

	// EventTypeSubscriptionCanceled is emitted after CancelSubscription.
	EventTypeSubscriptionCanceled = "billing.subscription.canceled"

	// EventTypeInvoiceCreated is emitted after CreateInvoice.
	EventTypeInvoiceCreated = "billing.invoice.created"
)

// SubscriptionEventPayload is the typed payload for subscription lifecycle events.
type SubscriptionEventPayload struct {
	SubscriptionID string    `json:"subscription_id"`
	CustomerID     string    `json:"customer_id"`
	PlanID         string    `json:"plan_id,omitempty"`
	Status         string    `json:"status,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// InvoiceEventPayload is the typed payload for invoice lifecycle events.
type InvoiceEventPayload struct {
	InvoiceID      string    `json:"invoice_id"`
	CustomerID     string    `json:"customer_id"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// EventedService decorates a Service to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so billing ops are not rolled back.
type EventedService struct {
	next Service
	bus  events.Bus
}

// NewEventedService wraps next so subscription/invoice mutations fan out to bus.
// If bus is nil, publishing is skipped.
func NewEventedService(next Service, bus events.Bus) *EventedService {
	return &EventedService{next: next, bus: bus}
}

func (s *EventedService) publish(ctx context.Context, eventType string, payload interface{}) {
	if s.bus == nil {
		return
	}
	_ = s.bus.Publish(ctx, TopicBilling, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/commerce/billing",
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// GetPlan delegates to the underlying service.
func (s *EventedService) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return s.next.GetPlan(ctx, planID)
}

// ListPlans delegates to the underlying service.
func (s *EventedService) ListPlans(ctx context.Context) ([]*Plan, error) {
	return s.next.ListPlans(ctx)
}

// CreateSubscription delegates then publishes billing.subscription.created.
func (s *EventedService) CreateSubscription(ctx context.Context, customerID string, planID string) (*Subscription, error) {
	sub, err := s.next.CreateSubscription(ctx, customerID, planID)
	if err != nil {
		return nil, err
	}
	s.publish(ctx, EventTypeSubscriptionCreated, SubscriptionEventPayload{
		SubscriptionID: sub.ID,
		CustomerID:     sub.CustomerID,
		PlanID:         sub.PlanID,
		Status:         string(sub.Status),
		Timestamp:      sub.CreatedAt,
	})
	return sub, nil
}

// CancelSubscription delegates then publishes billing.subscription.canceled.
func (s *EventedService) CancelSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	sub, err := s.next.CancelSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	s.publish(ctx, EventTypeSubscriptionCanceled, SubscriptionEventPayload{
		SubscriptionID: sub.ID,
		CustomerID:     sub.CustomerID,
		PlanID:         sub.PlanID,
		Status:         string(sub.Status),
		Timestamp:      time.Now().UTC(),
	})
	return sub, nil
}

// GetSubscription delegates to the underlying service.
func (s *EventedService) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	return s.next.GetSubscription(ctx, subscriptionID)
}

// UpgradeSubscription delegates without publishing (optional follow-up).
func (s *EventedService) UpgradeSubscription(ctx context.Context, subscriptionID string, newPlanID string) (*Subscription, error) {
	return s.next.UpgradeSubscription(ctx, subscriptionID, newPlanID)
}

// MarkPastDue delegates to the underlying service.
func (s *EventedService) MarkPastDue(ctx context.Context, subscriptionID string) (*Subscription, error) {
	return s.next.MarkPastDue(ctx, subscriptionID)
}

// ProcessDunning delegates to the underlying service.
func (s *EventedService) ProcessDunning(ctx context.Context, subscriptionID string) (*DunningResult, error) {
	return s.next.ProcessDunning(ctx, subscriptionID)
}

// CreateInvoice delegates then publishes billing.invoice.created.
func (s *EventedService) CreateInvoice(ctx context.Context, customerID string, amount commerce.Money) (*Invoice, error) {
	inv, err := s.next.CreateInvoice(ctx, customerID, amount)
	if err != nil {
		return nil, err
	}
	s.publish(ctx, EventTypeInvoiceCreated, InvoiceEventPayload{
		InvoiceID:      inv.ID,
		CustomerID:     inv.CustomerID,
		SubscriptionID: inv.SubscriptionID,
		Amount:         inv.Amount.Amount,
		Currency:       inv.Amount.Currency,
		Status:         inv.Status,
		Timestamp:      inv.IssuedAt,
	})
	return inv, nil
}

// ListInvoices delegates to the underlying service.
func (s *EventedService) ListInvoices(ctx context.Context, customerID string) ([]*Invoice, error) {
	return s.next.ListInvoices(ctx, customerID)
}
