package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/google/uuid"
)

// OutboxPayload is the JSON envelope published to messaging for a domain event.
type OutboxPayload struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload"`
}

// Outbox bridges in-process domain events to pkg/messaging for durable fan-out
// (PACKAGE_STANDARDS §9.5). It does not provide a transactional store; callers
// that need exactly-once should persist first, then call Publish.
type Outbox struct {
	producer messaging.Producer
}

// NewOutbox wraps a messaging.Producer.
func NewOutbox(producer messaging.Producer) *Outbox {
	return &Outbox{producer: producer}
}

// Publish serializes event and publishes it to the messaging producer.
// The bus topic is stored in the envelope and as the Message.Topic when empty
// on the producer default.
func (o *Outbox) Publish(ctx context.Context, topic string, event Event) error {
	if o == nil || o.producer == nil {
		return errors.InvalidArgument("outbox producer is required", nil)
	}
	if topic == "" {
		return ErrInvalidTopic(topic, nil)
	}
	if event.Type == "" {
		return ErrInvalidEvent("event type is required", nil)
	}

	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	raw, err := json.Marshal(event.Payload)
	if err != nil {
		return errors.Internal("failed to marshal event payload", err)
	}

	envelope, err := json.Marshal(OutboxPayload{
		ID:        event.ID,
		Type:      event.Type,
		Source:    event.Source,
		Timestamp: event.Timestamp,
		Topic:     topic,
		Payload:   raw,
	})
	if err != nil {
		return errors.Internal("failed to marshal outbox envelope", err)
	}

	msg := &messaging.Message{
		ID:      event.ID,
		Topic:   topic,
		Payload: envelope,
		Headers: map[string]string{
			"x-events-type":   event.Type,
			"x-events-source": event.Source,
			"x-events-topic":  topic,
		},
		Timestamp: event.Timestamp,
	}
	if err := o.producer.Publish(ctx, msg); err != nil {
		return errors.Wrap(err, "outbox publish failed")
	}
	return nil
}

// OutboxBus publishes to both a local Bus and a messaging Outbox.
type OutboxBus struct {
	Bus
	outbox *Outbox
}

// NewOutboxBus decorates bus so Publish also fans out via outbox.
func NewOutboxBus(bus Bus, outbox *Outbox) *OutboxBus {
	return &OutboxBus{Bus: bus, outbox: outbox}
}

// Publish delivers to the local bus then the messaging outbox.
func (b *OutboxBus) Publish(ctx context.Context, topic string, event Event) error {
	if err := b.Bus.Publish(ctx, topic, event); err != nil {
		return err
	}
	return b.outbox.Publish(ctx, topic, event)
}
