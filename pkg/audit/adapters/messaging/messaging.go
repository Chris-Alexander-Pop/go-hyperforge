package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/audit"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure compile-time interface compliance.
var (
	_ audit.Store          = (*FanoutStore)(nil)
	_ audit.LifecycleStore = (*FanoutStore)(nil)
)

// Config configures the messaging fanout store.
type Config struct {
	// Topic is the messaging topic for published audit events.
	Topic string

	// PublishOnly skips the inner store Append (messaging is the sole sink).
	// When false (default), Append writes to Inner first, then publishes.
	PublishOnly bool

	// Retrier wraps Publish; nil uses resilience.DefaultRetryConfig.
	Retrier resilience.Retrier
}

// FanoutStore decorates an audit.Store with messaging Append fanout.
type FanoutStore struct {
	inner       audit.Store
	producer    messaging.Producer
	topic       string
	publishOnly bool
	retrier     resilience.Retrier
}

// New creates a FanoutStore. Inner may be nil when PublishOnly is true.
func New(inner audit.Store, producer messaging.Producer, cfg Config) (*FanoutStore, error) {
	if producer == nil {
		return nil, audit.ErrInvalidArgument("producer is required", nil)
	}
	if cfg.Topic == "" {
		return nil, audit.ErrInvalidArgument("topic is required", nil)
	}
	if inner == nil && !cfg.PublishOnly {
		return nil, audit.ErrInvalidArgument("inner store is required unless PublishOnly", nil)
	}
	retrier := cfg.Retrier
	if retrier == nil {
		retrier = resilience.NewRetrier(resilience.DefaultRetryConfig())
	}
	return &FanoutStore{
		inner:       inner,
		producer:    producer,
		topic:       cfg.Topic,
		publishOnly: cfg.PublishOnly,
		retrier:     retrier,
	}, nil
}

// Append persists (unless PublishOnly) then publishes the event as JSON.
func (s *FanoutStore) Append(ctx context.Context, event audit.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.EventType == "" {
		return audit.ErrInvalidArgument("event_type is required", nil)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if !s.publishOnly {
		if err := s.inner.Append(ctx, event); err != nil {
			return err
		}
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return audit.ErrMarshalFailed(err)
	}

	msg := &messaging.Message{
		Topic:     s.topic,
		Key:       []byte(event.ActorID),
		Payload:   payload,
		Timestamp: event.Timestamp,
		Headers: map[string]string{
			"event_type": string(event.EventType),
			"outcome":    string(event.Outcome),
		},
	}

	return s.retrier.Execute(ctx, func(ctx context.Context) error {
		if err := s.producer.Publish(ctx, msg); err != nil {
			return audit.ErrAppendFailed("messaging publish failed", err)
		}
		return nil
	})
}

// Query delegates to the inner store.
func (s *FanoutStore) Query(ctx context.Context, filter audit.QueryFilter) ([]audit.Event, error) {
	if s.inner == nil {
		return nil, audit.ErrNotSupported
	}
	return s.inner.Query(ctx, filter)
}

// Purge delegates to the inner LifecycleStore when available.
func (s *FanoutStore) Purge(ctx context.Context, olderThan time.Time) (int64, error) {
	ls, ok := s.inner.(audit.RetentionStore)
	if !ok || s.inner == nil {
		return 0, audit.ErrNotSupported
	}
	return ls.Purge(ctx, olderThan)
}

// ExportByActor delegates to the inner PrivacyStore when available.
func (s *FanoutStore) ExportByActor(ctx context.Context, actorID string) ([]audit.Event, error) {
	ps, ok := s.inner.(audit.PrivacyStore)
	if !ok || s.inner == nil {
		return nil, audit.ErrNotSupported
	}
	return ps.ExportByActor(ctx, actorID)
}

// EraseByActor delegates to the inner PrivacyStore when available.
func (s *FanoutStore) EraseByActor(ctx context.Context, actorID string) (int64, error) {
	ps, ok := s.inner.(audit.PrivacyStore)
	if !ok || s.inner == nil {
		return 0, audit.ErrNotSupported
	}
	return ps.EraseByActor(ctx, actorID)
}
