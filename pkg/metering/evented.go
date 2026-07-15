package metering

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedMeter implements Meter at compile time.
var _ Meter = (*EventedMeter)(nil)

// UsageRecordedPayload is the typed payload for metering.usage.recorded events.
type UsageRecordedPayload struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Quantity     float64   `json:"quantity"`
	Timestamp    time.Time `json:"timestamp"`
}

const (
	// TopicUsage is the pkg/events topic for metering usage domain events.
	TopicUsage = "metering"

	// EventTypeUsageRecorded is emitted after a successful RecordUsage.
	EventTypeUsageRecorded = "metering.usage.recorded"
)

// EventedMeter decorates a Meter to publish domain events after successful RecordUsage.
// Publish is best-effort: failures are ignored so recording is not rolled back.
type EventedMeter struct {
	next Meter
	bus  events.Bus
}

// NewEventedMeter wraps next so RecordUsage fans out to bus after persistence.
// If bus is nil, publishing is skipped and operations still delegate to next.
func NewEventedMeter(next Meter, bus events.Bus) *EventedMeter {
	return &EventedMeter{next: next, bus: bus}
}

// RecordUsage records usage then publishes metering.usage.recorded (best-effort).
func (m *EventedMeter) RecordUsage(ctx context.Context, event UsageEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if err := m.next.RecordUsage(ctx, event); err != nil {
		return err
	}
	if m.bus == nil {
		return nil
	}

	_ = m.bus.Publish(ctx, TopicUsage, events.Event{
		ID:        event.ID,
		Type:      EventTypeUsageRecorded,
		Source:    "pkg/metering",
		Timestamp: event.Timestamp,
		Payload: UsageRecordedPayload{
			ID:           event.ID,
			TenantID:     event.TenantID,
			ResourceType: event.ResourceType,
			ResourceID:   event.ResourceID,
			Quantity:     event.Quantity,
			Timestamp:    event.Timestamp,
		},
	})
	return nil
}

// GetUsage delegates to the underlying meter.
func (m *EventedMeter) GetUsage(ctx context.Context, filter UsageFilter) ([]UsageEvent, error) {
	return m.next.GetUsage(ctx, filter)
}

// Close delegates to the underlying meter.
func (m *EventedMeter) Close() error {
	return m.next.Close()
}
