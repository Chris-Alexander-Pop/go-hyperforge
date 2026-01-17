// Package events provides an in-process event bus for domain events.
//
// This package is for local pub/sub within a single process.
// For distributed messaging, use pkg/messaging instead.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/events/adapters/memory"
//
//	bus := memory.New()
//	bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error {
//		// Handle event
//	})
//	bus.Publish(ctx, "users", events.Event{Type: "user.created", Payload: user})
package events

import (
	"context"
	"time"
)

// Event represents a standard event (inspired by CloudEvents)
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`   // e.g. "user.created"
	Source    string      `json:"source"` // e.g. "user-service"
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Handler handles an incoming event
type Handler func(ctx context.Context, event Event) error

// Bus defines the interface for an event bus
type Bus interface {
	Publish(ctx context.Context, topic string, event Event) error
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Close() error
}
