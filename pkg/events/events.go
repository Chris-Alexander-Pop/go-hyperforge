package events

import (
	"context"
	"time"
)

// Config holds event bus configuration.
type Config struct {
	// Async dispatches handlers via a bounded worker pool instead of synchronously.
	// When false (default), Publish runs handlers in-process and returns aggregated errors.
	Async bool `env:"EVENTS_ASYNC" env-default:"false"`

	// Workers is the number of worker-pool goroutines when Async is true.
	Workers int `env:"EVENTS_WORKERS" env-default:"4"`

	// QueueSize is the worker-pool task queue capacity when Async is true.
	QueueSize int `env:"EVENTS_QUEUE_SIZE" env-default:"256"`
}

// DefaultConfig returns Config with package defaults applied.
func DefaultConfig() Config {
	return Config{
		Async:     false,
		Workers:   4,
		QueueSize: 256,
	}
}

// Event represents a standard event (inspired by CloudEvents).
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`   // e.g. "user.created"
	Source    string      `json:"source"` // e.g. "user-service"
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Handler handles an incoming event.
type Handler func(ctx context.Context, event Event) error

// Subscription identifies an active topic subscription.
type Subscription string

// Bus defines the interface for an event bus.
type Bus interface {
	Publish(ctx context.Context, topic string, event Event) error
	Subscribe(ctx context.Context, topic string, handler Handler) (Subscription, error)
	Unsubscribe(ctx context.Context, id Subscription) error
	Close() error
}
