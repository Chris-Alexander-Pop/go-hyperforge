package blob

import (
	"context"
	"io"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedStore implements Store.
var _ Store = (*EventedStore)(nil)

const (
	// TopicBlob is the pkg/events topic for blob domain events.
	TopicBlob = "blob"

	// EventTypeUploaded is emitted after a successful Upload.
	EventTypeUploaded = "blob.uploaded"

	// EventTypeDeleted is emitted after a successful Delete.
	EventTypeDeleted = "blob.deleted"
)

// BlobEventPayload is the typed payload for blob lifecycle events.
type BlobEventPayload struct {
	Key string `json:"key"`
}

// EventedStore decorates a Store to emit events via pkg/events.
// Publish is best-effort: failures are ignored so storage writes are not rolled back.
type EventedStore struct {
	next Store
	bus  events.Bus
}

// NewEventedStore wraps next so Upload/Delete fan out to bus.
// If bus is nil, publishing is skipped and operations still delegate to next.
func NewEventedStore(next Store, bus events.Bus) *EventedStore {
	return &EventedStore{next: next, bus: bus}
}

func (s *EventedStore) publish(ctx context.Context, eventType, key string) {
	if s.bus == nil {
		return
	}
	id := key
	if id == "" {
		id = uuid.NewString()
	}
	_ = s.bus.Publish(ctx, TopicBlob, events.Event{
		ID:        id + ":" + eventType,
		Type:      eventType,
		Source:    "pkg/storage/blob",
		Timestamp: time.Now().UTC(),
		Payload: BlobEventPayload{
			Key: key,
		},
	})
}

func (s *EventedStore) Upload(ctx context.Context, key string, data io.Reader) error {
	err := s.next.Upload(ctx, key, data)
	if err == nil {
		s.publish(ctx, EventTypeUploaded, key)
	}
	return err
}

func (s *EventedStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.next.Download(ctx, key)
}

func (s *EventedStore) Delete(ctx context.Context, key string) error {
	err := s.next.Delete(ctx, key)
	if err == nil {
		s.publish(ctx, EventTypeDeleted, key)
	}
	return err
}

func (s *EventedStore) URL(key string) string {
	return s.next.URL(key)
}
