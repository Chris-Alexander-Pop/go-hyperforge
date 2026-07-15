package secrets

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/google/uuid"
)

const (
	// TopicSecrets is the pkg/events topic for secret domain events.
	TopicSecrets = "secrets"

	// EventTypeSecretSet is emitted after a successful Set (best-effort).
	EventTypeSecretSet = "secrets.set"

	// EventTypeSecretRotated is emitted after a successful Rotate (best-effort).
	EventTypeSecretRotated = "secrets.rotated"
)

// SecretAuditPayload is a redaction-safe payload for secret lifecycle events.
// It never includes the secret value.
type SecretAuditPayload struct {
	Name      string    `json:"name"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
}

// EventedSecretManager decorates a SecretManager to publish audit-friendly events.
// Publish is best-effort: failures are ignored so secret ops are not rolled back.
type EventedSecretManager struct {
	next SecretManager
	bus  events.Bus
}

// Ensure EventedSecretManager implements SecretManager.
var _ SecretManager = (*EventedSecretManager)(nil)

// NewEventedSecretManager wraps next so Set/Rotate fan out to bus after success.
// If bus is nil, publishing is skipped.
func NewEventedSecretManager(next SecretManager, bus events.Bus) *EventedSecretManager {
	return &EventedSecretManager{next: next, bus: bus}
}

func (m *EventedSecretManager) Get(ctx context.Context, name string) (string, error) {
	return m.next.Get(ctx, name)
}

func (m *EventedSecretManager) Set(ctx context.Context, name, value string) error {
	if err := m.next.Set(ctx, name, value); err != nil {
		return err
	}
	m.publish(ctx, EventTypeSecretSet, name)
	return nil
}

func (m *EventedSecretManager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	val, err := m.next.Rotate(ctx, name, newValue)
	if err != nil {
		return "", err
	}
	m.publish(ctx, EventTypeSecretRotated, name)
	return val, nil
}

func (m *EventedSecretManager) publish(ctx context.Context, eventType, name string) {
	if m.bus == nil {
		return
	}
	now := time.Now().UTC()
	_ = m.bus.Publish(ctx, TopicSecrets, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/security/secrets",
		Timestamp: now,
		Payload: SecretAuditPayload{
			Name:      name,
			Operation: eventType,
			Timestamp: now,
		},
	})
}
