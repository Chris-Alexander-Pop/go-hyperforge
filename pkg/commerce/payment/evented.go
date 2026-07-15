package payment

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedProvider implements Provider at compile time.
var _ Provider = (*EventedProvider)(nil)

const (
	// TopicPayment is the pkg/events topic for payment domain events.
	TopicPayment = "commerce.payment"

	// EventTypeChargeSucceeded is emitted after a successful Charge.
	EventTypeChargeSucceeded = "payment.charge.succeeded"

	// EventTypeRefundSucceeded is emitted after a successful Refund.
	EventTypeRefundSucceeded = "payment.refund.succeeded"
)

// ChargeSucceededPayload is the typed payload for payment.charge.succeeded.
type ChargeSucceededPayload struct {
	TransactionID  string    `json:"transaction_id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	SourceID       string    `json:"source_id"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// RefundSucceededPayload is the typed payload for payment.refund.succeeded.
type RefundSucceededPayload struct {
	TransactionID       string    `json:"transaction_id"`
	OriginalTransaction string    `json:"original_transaction"`
	Amount              int64     `json:"amount"`
	Currency            string    `json:"currency"`
	Timestamp           time.Time `json:"timestamp"`
}

// EventedProvider decorates a Provider to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so payment is not rolled back.
type EventedProvider struct {
	next Provider
	bus  events.Bus
}

// NewEventedProvider wraps next so Charge/Refund fan out to bus after success.
// If bus is nil, publishing is skipped and operations still delegate to next.
func NewEventedProvider(next Provider, bus events.Bus) *EventedProvider {
	return &EventedProvider{next: next, bus: bus}
}

func (p *EventedProvider) publish(ctx context.Context, eventType string, payload interface{}) {
	if p.bus == nil {
		return
	}
	_ = p.bus.Publish(ctx, TopicPayment, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/commerce/payment",
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// Charge delegates then publishes payment.charge.succeeded (best-effort).
func (p *EventedProvider) Charge(ctx context.Context, req *ChargeRequest) (*Transaction, error) {
	tx, err := p.next.Charge(ctx, req)
	if err != nil {
		return nil, err
	}
	p.publish(ctx, EventTypeChargeSucceeded, ChargeSucceededPayload{
		TransactionID:  tx.ID,
		Amount:         tx.Amount.Amount,
		Currency:       tx.Amount.Currency,
		SourceID:       tx.SourceID,
		IdempotencyKey: tx.IdempotencyKey,
		Timestamp:      tx.CreatedAt,
	})
	return tx, nil
}

// Refund delegates then publishes payment.refund.succeeded (best-effort).
func (p *EventedProvider) Refund(ctx context.Context, req *RefundRequest) (*Transaction, error) {
	tx, err := p.next.Refund(ctx, req)
	if err != nil {
		return nil, err
	}
	p.publish(ctx, EventTypeRefundSucceeded, RefundSucceededPayload{
		TransactionID:       tx.ID,
		OriginalTransaction: req.TransactionID,
		Amount:              tx.Amount.Amount,
		Currency:            tx.Amount.Currency,
		Timestamp:           tx.CreatedAt,
	})
	return tx, nil
}

// GetTransaction delegates to the underlying provider.
func (p *EventedProvider) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	return p.next.GetTransaction(ctx, id)
}

// Close delegates to the underlying provider.
func (p *EventedProvider) Close() error {
	return p.next.Close()
}
