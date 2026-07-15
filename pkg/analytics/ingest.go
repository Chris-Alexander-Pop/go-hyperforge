package analytics

import (
	"context"
	"time"
)

// Event is a single analytics event for ingest sinks.
type Event struct {
	// Name is the event type (e.g. "page_view", "purchase").
	Name string `json:"name"`

	// UserID optionally identifies the actor.
	UserID string `json:"user_id,omitempty"`

	// Properties holds arbitrary event attributes.
	Properties map[string]any `json:"properties,omitempty"`

	// Timestamp is when the event occurred. Zero means "now" at ingest time.
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// Sink ingests analytics events (warehouse / streaming style).
// This is separate from Tracker (HyperLogLog uniqueness).
type Sink interface {
	// Ingest appends one or more events. An empty slice is a no-op.
	Ingest(ctx context.Context, events ...Event) error

	// Close releases sink resources. After Close, Ingest returns ErrClosed.
	Close() error
}

// WindowKey returns a time-bucketed counter name for windowed uniqueness.
// Example: WindowKey("visitors", t, time.Hour) → "visitors:2026-07-15T05".
// window <= 0 uses a 1-hour bucket.
func WindowKey(counter string, t time.Time, window time.Duration) string {
	if window <= 0 {
		window = time.Hour
	}
	if t.IsZero() {
		t = time.Now().UTC()
	} else {
		t = t.UTC()
	}
	sec := int64(window / time.Second)
	if sec <= 0 {
		sec = 1
	}
	unix := t.Unix()
	bucket := unix - (unix % sec)
	bt := time.Unix(bucket, 0).UTC()
	switch {
	case window >= 24*time.Hour:
		return counter + ":" + bt.Format("2006-01-02")
	case window >= time.Hour:
		return counter + ":" + bt.Format("2006-01-02T15")
	default:
		return counter + ":" + bt.Format("2006-01-02T15:04")
	}
}

// WindowedUniqueness records uniqueness against time-bucketed Tracker counters.
type WindowedUniqueness struct {
	tracker Tracker
	window  time.Duration
	counter string
}

// NewWindowedUniqueness wraps a Tracker for windowed unique counts.
// window <= 0 defaults to one hour.
func NewWindowedUniqueness(tracker Tracker, counter string, window time.Duration) *WindowedUniqueness {
	if window <= 0 {
		window = time.Hour
	}
	return &WindowedUniqueness{tracker: tracker, window: window, counter: counter}
}

// Add records element in the current (or provided) time window.
func (w *WindowedUniqueness) Add(ctx context.Context, element string, at time.Time) error {
	if w == nil || w.tracker == nil {
		return ErrClosed
	}
	key := WindowKey(w.counter, at, w.window)
	return w.tracker.Add(ctx, key, element)
}

// Count returns the estimated unique count for the window containing at.
func (w *WindowedUniqueness) Count(ctx context.Context, at time.Time) (uint64, error) {
	if w == nil || w.tracker == nil {
		return 0, ErrClosed
	}
	key := WindowKey(w.counter, at, w.window)
	return w.tracker.Count(ctx, key)
}
