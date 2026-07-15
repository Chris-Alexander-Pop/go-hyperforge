package memory

import (
	"context"
	"sync/atomic"

	"github.com/chris-alexander-pop/system-design-library/pkg/analytics"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/hyperloglog"
)

// Ensure Tracker implements analytics.Tracker.
var _ analytics.Tracker = (*Tracker)(nil)

// Tracker implements an in-memory analytics tracker using HyperLogLog.
type Tracker struct {
	counters  map[string]*hyperloglog.HyperLogLog
	precision uint8
	mu        *concurrency.SmartRWMutex
	closed    atomic.Bool
}

// New creates a new in-memory tracker.
// Config is normalized and validated via pkg/validator (Precision must be 4–16).
func New(cfg analytics.Config) (analytics.Tracker, error) {
	cfg, err := cfg.Normalize()
	if err != nil {
		return nil, err
	}

	return &Tracker{
		counters:  make(map[string]*hyperloglog.HyperLogLog),
		precision: cfg.Precision,
		mu:        concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "AnalyticsTracker"}),
	}, nil
}

func (t *Tracker) guard() error {
	if t.closed.Load() {
		return analytics.ErrClosed
	}
	return nil
}

// Add records an element for the given counter name.
func (t *Tracker) Add(ctx context.Context, counterName string, element string) error {
	if err := t.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	counter, exists := t.counters[counterName]
	if !exists {
		counter = hyperloglog.New(t.precision)
		t.counters[counterName] = counter
	}
	counter.AddString(element)
	return nil
}

// Count returns the estimated unique count for the given counter.
// Missing counters return (0, nil).
func (t *Tracker) Count(ctx context.Context, counterName string) (uint64, error) {
	if err := t.guard(); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	counter, exists := t.counters[counterName]
	if !exists {
		return 0, nil
	}
	return counter.Count(), nil
}

// Reset clears a specific counter. Missing counters are a no-op.
func (t *Tracker) Reset(ctx context.Context, counterName string) error {
	if err := t.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if counter, exists := t.counters[counterName]; exists {
		counter.Clear()
	}
	return nil
}

// Merge merges the HyperLogLog sketch of source into dest.
// Returns analytics.ErrCounterNotFound if source does not exist.
func (t *Tracker) Merge(ctx context.Context, dest, source string) error {
	if err := t.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	src, ok := t.counters[source]
	if !ok {
		return analytics.ErrCounterNotFound
	}

	dst, ok := t.counters[dest]
	if !ok {
		dst = hyperloglog.New(t.precision)
		t.counters[dest] = dst
	}

	if !dst.Merge(src) {
		// Same-tracker sketches always share precision; treat as internal failure.
		return analytics.ErrCounterNotFound
	}
	return nil
}

// Close releases tracker resources. Subsequent operations return ErrClosed.
func (t *Tracker) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counters = nil
	return nil
}
