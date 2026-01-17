// Package analytics provides utilities for tracking and measuring application metrics.
//
// This package includes:
//   - UniqueCounter: HyperLogLog-based unique element counting
//   - Use cases: unique visitors, unique errors, cardinality estimation
package analytics

import (
	"sync"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/hyperloglog"
)

// UniqueCounter tracks unique elements using HyperLogLog.
// Memory-efficient counting of distinct items with configurable precision.
//
// Example use cases:
//   - Count unique visitors per day
//   - Count unique search queries
//   - Count unique error types
type UniqueCounter struct {
	counters  map[string]*hyperloglog.HyperLogLog
	precision uint8
	mu        sync.RWMutex
}

// NewUniqueCounter creates a new unique counter with the given precision.
// Precision 10-14 is recommended (1KB-16KB memory per counter, 0.8%-3% error).
func NewUniqueCounter(precision uint8) *UniqueCounter {
	return &UniqueCounter{
		counters:  make(map[string]*hyperloglog.HyperLogLog),
		precision: precision,
	}
}

// Add records an element for the given counter name.
func (uc *UniqueCounter) Add(counterName string, element string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	counter, exists := uc.counters[counterName]
	if !exists {
		counter = hyperloglog.New(uc.precision)
		uc.counters[counterName] = counter
	}
	counter.AddString(element)
}

// Count returns the estimated unique count for the given counter.
func (uc *UniqueCounter) Count(counterName string) uint64 {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	counter, exists := uc.counters[counterName]
	if !exists {
		return 0
	}
	return counter.Count()
}

// Merge combines another counter into this one.
// Useful for aggregating counts from multiple sources.
func (uc *UniqueCounter) Merge(counterName string, other *hyperloglog.HyperLogLog) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	counter, exists := uc.counters[counterName]
	if !exists {
		counter = hyperloglog.New(uc.precision)
		uc.counters[counterName] = counter
	}
	return counter.Merge(other)
}

// Reset clears a specific counter.
func (uc *UniqueCounter) Reset(counterName string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if counter, exists := uc.counters[counterName]; exists {
		counter.Clear()
	}
}

// ResetAll clears all counters.
func (uc *UniqueCounter) ResetAll() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.counters = make(map[string]*hyperloglog.HyperLogLog)
}

// Stats returns statistics for all counters.
func (uc *UniqueCounter) Stats() map[string]uint64 {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	result := make(map[string]uint64, len(uc.counters))
	for name, counter := range uc.counters {
		result[name] = counter.Count()
	}
	return result
}

// CounterNames returns the names of all active counters.
func (uc *UniqueCounter) CounterNames() []string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	names := make([]string, 0, len(uc.counters))
	for name := range uc.counters {
		names = append(names, name)
	}
	return names
}
