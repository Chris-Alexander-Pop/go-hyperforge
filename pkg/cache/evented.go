package cache

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedCache implements Cache at compile time.
var _ Cache = (*EventedCache)(nil)

const (
	// TopicCache is the pkg/events topic for cache domain events.
	TopicCache = "cache"

	// EventTypeSet is emitted after a successful Set.
	EventTypeSet = "cache.set"

	// EventTypeDeleted is emitted after a successful Delete.
	EventTypeDeleted = "cache.deleted"

	// EventTypeMSet is emitted after a successful MSet.
	EventTypeMSet = "cache.mset"
)

// CacheEventPayload is the typed payload for cache mutation events.
type CacheEventPayload struct {
	Key       string    `json:"key,omitempty"`
	Keys      []string  `json:"keys,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EventedCache decorates a Cache to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so cache writes are not rolled back.
type EventedCache struct {
	next Cache
	bus  events.Bus
}

// NewEventedCache wraps next so Set/Delete/MSet fan out to bus after success.
// If bus is nil, publishing is skipped.
func NewEventedCache(next Cache, bus events.Bus) *EventedCache {
	return &EventedCache{next: next, bus: bus}
}

func (c *EventedCache) publish(ctx context.Context, eventType, key string, keys []string) {
	if c.bus == nil {
		return
	}
	_ = c.bus.Publish(ctx, TopicCache, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/cache",
		Timestamp: time.Now().UTC(),
		Payload: CacheEventPayload{
			Key:       key,
			Keys:      keys,
			Timestamp: time.Now().UTC(),
		},
	})
}

// Get delegates to the underlying cache.
func (c *EventedCache) Get(ctx context.Context, key string, dest interface{}) error {
	return c.next.Get(ctx, key, dest)
}

// Set delegates then publishes cache.set (best-effort).
func (c *EventedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := c.next.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	c.publish(ctx, EventTypeSet, key, nil)
	return nil
}

// Delete delegates then publishes cache.deleted (best-effort).
func (c *EventedCache) Delete(ctx context.Context, key string) error {
	if err := c.next.Delete(ctx, key); err != nil {
		return err
	}
	c.publish(ctx, EventTypeDeleted, key, nil)
	return nil
}

// Exists delegates to the underlying cache.
func (c *EventedCache) Exists(ctx context.Context, key string) (bool, error) {
	return c.next.Exists(ctx, key)
}

// MGet delegates to the underlying cache.
func (c *EventedCache) MGet(ctx context.Context, keys []string, dest interface{}) error {
	return c.next.MGet(ctx, keys, dest)
}

// MSet delegates then publishes cache.mset (best-effort).
func (c *EventedCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if err := c.next.MSet(ctx, items, ttl); err != nil {
		return err
	}
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	c.publish(ctx, EventTypeMSet, "", keys)
	return nil
}

// Expire delegates to the underlying cache.
func (c *EventedCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.next.Expire(ctx, key, ttl)
}

// GetTTL delegates to the underlying cache.
func (c *EventedCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return c.next.GetTTL(ctx, key)
}

// Incr delegates to the underlying cache.
func (c *EventedCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return c.next.Incr(ctx, key, delta)
}

// Close releases resources held by the underlying cache.
func (c *EventedCache) Close() error {
	return c.next.Close()
}
