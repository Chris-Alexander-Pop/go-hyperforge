package memory

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Ensure Sink implements analytics.Sink.
var _ analytics.Sink = (*Sink)(nil)

// Sink is an in-memory analytics event sink for tests and local use.
type Sink struct {
	mu     *concurrency.SmartMutex
	events []analytics.Event
	closed atomic.Bool
}

// NewSink creates an empty in-memory event sink.
func NewSink() *Sink {
	return &Sink{
		mu:     concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "AnalyticsSink"}),
		events: make([]analytics.Event, 0),
	}
}

// Ingest appends events to the in-memory buffer.
func (s *Sink) Ingest(ctx context.Context, events ...analytics.Event) error {
	if s.closed.Load() {
		return analytics.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range events {
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now().UTC()
		}
		if e.Properties != nil {
			props := make(map[string]any, len(e.Properties))
			for k, v := range e.Properties {
				props[k] = v
			}
			e.Properties = props
		}
		s.events = append(s.events, e)
	}
	return nil
}

// Events returns a copy of ingested events (test helper).
func (s *Sink) Events() []analytics.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]analytics.Event, len(s.events))
	copy(out, s.events)
	return out
}

// Close marks the sink closed.
func (s *Sink) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	return nil
}
