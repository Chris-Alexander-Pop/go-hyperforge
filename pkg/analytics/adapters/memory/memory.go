package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/analytics"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/hyperloglog"
)

// Tracker implements an in-memory analytics tracker using HyperLogLog.
type Tracker struct {
	counters  map[string]*hyperloglog.HyperLogLog
	precision uint8
	mu        *concurrency.SmartRWMutex
}

// New creates a new in-memory tracker.
func New(cfg analytics.Config) analytics.Tracker {
	// Validate precision or set default
	if cfg.Precision < 4 || cfg.Precision > 18 {
		cfg.Precision = 14
	}

	return &Tracker{
		counters:  make(map[string]*hyperloglog.HyperLogLog),
		precision: cfg.Precision,
		mu:        concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "AnalyticsTracker"}),
	}
}

// Add records an element for the given counter name.
func (t *Tracker) Add(ctx context.Context, counterName string, element string) error {
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
func (t *Tracker) Count(ctx context.Context, counterName string) (uint64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	counter, exists := t.counters[counterName]
	if !exists {
		return 0, nil // Or ErrCounterNotFound, but 0 is usually safer for analytics
	}
	return counter.Count(), nil
}

// Reset clears a specific counter.
func (t *Tracker) Reset(ctx context.Context, counterName string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if counter, exists := t.counters[counterName]; exists {
		counter.Clear()
	}
	return nil
}
