package session

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedManager implements Manager at compile time.
var _ Manager = (*EventedManager)(nil)

const (
	// TopicSession is the pkg/events topic for session domain events.
	TopicSession = "auth.session"

	// EventTypeSessionCreated is emitted after a successful Create.
	EventTypeSessionCreated = "session.created"

	// EventTypeSessionDeleted is emitted after a successful Delete.
	EventTypeSessionDeleted = "session.deleted"

	// EventTypeSessionRefreshed is emitted after a successful Refresh.
	EventTypeSessionRefreshed = "session.refreshed"
)

// SessionEventPayload is the typed payload for session lifecycle events.
type SessionEventPayload struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EventedManager decorates a Manager to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so session ops are not rolled back.
type EventedManager struct {
	next Manager
	bus  events.Bus
}

// NewEventedManager wraps next so Create/Delete/Refresh fan out to bus after success.
// If bus is nil, publishing is skipped.
func NewEventedManager(next Manager, bus events.Bus) *EventedManager {
	return &EventedManager{next: next, bus: bus}
}

func (m *EventedManager) publish(ctx context.Context, eventType, sessionID, userID string) {
	if m.bus == nil {
		return
	}
	_ = m.bus.Publish(ctx, TopicSession, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/auth/session",
		Timestamp: time.Now().UTC(),
		Payload: SessionEventPayload{
			SessionID: sessionID,
			UserID:    userID,
			Timestamp: time.Now().UTC(),
		},
	})
}

// Create delegates then publishes session.created (best-effort).
func (m *EventedManager) Create(ctx context.Context, userID string, metadata map[string]interface{}) (*Session, error) {
	s, err := m.next.Create(ctx, userID, metadata)
	if err != nil {
		return nil, err
	}
	m.publish(ctx, EventTypeSessionCreated, s.ID, s.UserID)
	return s, nil
}

// Get delegates to the underlying manager.
func (m *EventedManager) Get(ctx context.Context, sessionID string) (*Session, error) {
	return m.next.Get(ctx, sessionID)
}

// Delete delegates then publishes session.deleted (best-effort).
func (m *EventedManager) Delete(ctx context.Context, sessionID string) error {
	if err := m.next.Delete(ctx, sessionID); err != nil {
		return err
	}
	m.publish(ctx, EventTypeSessionDeleted, sessionID, "")
	return nil
}

// Refresh delegates then publishes session.refreshed (best-effort).
func (m *EventedManager) Refresh(ctx context.Context, sessionID string) (*Session, error) {
	s, err := m.next.Refresh(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	m.publish(ctx, EventTypeSessionRefreshed, s.ID, s.UserID)
	return s, nil
}
