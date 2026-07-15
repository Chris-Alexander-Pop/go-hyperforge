package memory

import (
	"context"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Ensure ExactStore implements analytics.CounterStore.
var _ analytics.CounterStore = (*ExactStore)(nil)

// ExactStore is an in-memory exact (non-HLL) CounterStore.
//
// Two modes share the same named counters map:
//   - Incr maintains an int64 total
//   - AddExact maintains a set of unique elements; Count prefers set size when present
type ExactStore struct {
	totals map[string]int64
	sets   map[string]map[string]struct{}
	mu     *concurrency.SmartRWMutex
	closed atomic.Bool
}

// NewExact creates an exact CounterStore (non-HyperLogLog).
func NewExact() *ExactStore {
	return &ExactStore{
		totals: make(map[string]int64),
		sets:   make(map[string]map[string]struct{}),
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "AnalyticsExactStore"}),
	}
}

func (s *ExactStore) guard() error {
	if s.closed.Load() {
		return analytics.ErrClosed
	}
	return nil
}

// Incr increments a named counter by delta.
func (s *ExactStore) Incr(ctx context.Context, counter string, delta int64) (int64, error) {
	if err := s.guard(); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totals[counter] += delta
	return s.totals[counter], nil
}

// AddExact records a unique element for exact set cardinality.
func (s *ExactStore) AddExact(ctx context.Context, counter string, element string) error {
	if err := s.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	set := s.sets[counter]
	if set == nil {
		set = make(map[string]struct{})
		s.sets[counter] = set
	}
	set[element] = struct{}{}
	return nil
}

// Count returns exact total: set size if AddExact was used, otherwise Incr total.
func (s *ExactStore) Count(ctx context.Context, counter string) (int64, error) {
	if err := s.guard(); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if set, ok := s.sets[counter]; ok {
		return int64(len(set)), nil
	}
	return s.totals[counter], nil
}

// Reset clears both total and set for a counter.
func (s *ExactStore) Reset(ctx context.Context, counter string) error {
	if err := s.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.totals, counter)
	delete(s.sets, counter)
	return nil
}

// Close releases resources.
func (s *ExactStore) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totals = nil
	s.sets = nil
	return nil
}
