// Package concurrency provides advanced concurrency primitives with observability.
//
// This package includes:
//   - SmartMutex: sync.Mutex with slow lock detection and deadlock monitoring
//   - SmartRWMutex: sync.RWMutex with the same observability features
//   - Semaphore: Counting semaphore for resource limiting
//   - WorkerPool: Managed pool of worker goroutines
//   - Pipeline: Concurrent data processing pipeline
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
//
//	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "my-mutex"})
//	mu.Lock()
//	defer mu.Unlock()
package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// MutexConfig controls the behavior of SmartMutex and SmartRWMutex.
type MutexConfig struct {
	// Name identifies this mutex in logs (only used in DebugMode).
	Name string

	// SlowThreshold logs a warning if the lock is held longer than this (only in DebugMode).
	// Default: 100ms
	SlowThreshold time.Duration

	// MonitorInterval is how often to check for potential deadlocks (only in DebugMode).
	MonitorInterval time.Duration

	// DebugMode enables observability features: caller tracking, duration logging,
	// and slow lock detection. This adds ~1000ns overhead per Lock() due to
	// runtime.Caller(). Use this when debugging lock contention issues.
	//
	// Default is false (fast mode) for production performance.
	DebugMode bool
}

// SmartMutex is a sync.Mutex with observability
type SmartMutex struct {
	mu       sync.Mutex
	config   MutexConfig
	holder   atomic.Value // Stores string (stack trace or caller)
	lockedAt atomic.Int64 // UnixMilli
	isLocked atomic.Bool
}

func NewSmartMutex(cfg MutexConfig) *SmartMutex {
	if cfg.SlowThreshold == 0 {
		cfg.SlowThreshold = 100 * time.Millisecond
	}
	return &SmartMutex{config: cfg}
}

func (m *SmartMutex) Lock() {
	m.mu.Lock()

	// Only track in DebugMode (expensive observability)
	if !m.config.DebugMode {
		return
	}

	// Record State
	m.lockedAt.Store(time.Now().UnixMilli())
	m.isLocked.Store(true)

	// Debug info (Caller) - expensive but useful for debugging
	_, file, line, ok := runtime.Caller(1)
	if ok {
		m.holder.Store(fmt.Sprintf("%s:%d", file, line))
	}
}

func (m *SmartMutex) Unlock() {
	// Fast path: no observability
	if !m.config.DebugMode {
		m.mu.Unlock()
		return
	}

	// Check Duration
	start := m.lockedAt.Load()
	duration := time.Since(time.UnixMilli(start))

	holder := m.holder.Load()

	m.isLocked.Store(false)
	m.mu.Unlock()

	// Log if slow
	if duration > m.config.SlowThreshold {
		logger.L().Warn("SmartMutex held too long",
			"name", m.config.Name,
			"duration", duration,
			"caller", holder,
		)
	}
}

// SmartRWMutex is a sync.RWMutex with observability
type SmartRWMutex struct {
	mu       sync.RWMutex
	config   MutexConfig
	holder   atomic.Value
	lockedAt atomic.Int64
	isLocked atomic.Bool // Only tracks Write locks for simplicity/deadlock risk
}

func NewSmartRWMutex(cfg MutexConfig) *SmartRWMutex {
	if cfg.SlowThreshold == 0 {
		cfg.SlowThreshold = 100 * time.Millisecond
	}
	return &SmartRWMutex{config: cfg}
}

func (m *SmartRWMutex) Lock() {
	m.mu.Lock()

	// Only track in DebugMode (expensive observability)
	if !m.config.DebugMode {
		return
	}

	m.lockedAt.Store(time.Now().UnixMilli())
	m.isLocked.Store(true)
	_, file, line, ok := runtime.Caller(1)
	if ok {
		m.holder.Store(fmt.Sprintf("%s:%d", file, line))
	}
}

func (m *SmartRWMutex) Unlock() {
	// Fast path: no observability
	if !m.config.DebugMode {
		m.mu.Unlock()
		return
	}

	// Check duration logic same as Mutex
	start := m.lockedAt.Load()
	duration := time.Since(time.UnixMilli(start))
	holder := m.holder.Load()
	m.isLocked.Store(false)
	m.mu.Unlock()
	if duration > m.config.SlowThreshold {
		logger.L().Warn("SmartRWMutex Write held too long", "name", m.config.Name, "duration", duration, "caller", holder)
	}
}

func (m *SmartRWMutex) RLock() {
	m.mu.RLock()
	// Read locks are harder to track for deadlock (multiple holders).
	// We skip tracking individual read hold times for now to avoid overhead/complexity.
}

func (m *SmartRWMutex) RUnlock() {
	m.mu.RUnlock()
}

// Monitor spawns a background routine to watch this mutex (Simulated Deadlock Detector)
func (m *SmartMutex) Monitor(ctx context.Context) {
	ticker := time.NewTicker(m.config.SlowThreshold * 2) // Check sparingly
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.isLocked.Load() {
				start := m.lockedAt.Load()
				if time.Since(time.UnixMilli(start)) > m.config.SlowThreshold*5 {
					// Serious deadlock risk
					logger.L().ErrorContext(ctx, "POTENTIAL DEADLOCK: SmartMutex held excessively long",
						"name", m.config.Name,
						"duration", time.Since(time.UnixMilli(start)),
						"holder_caller", m.holder.Load(),
					)
				}
			}
		}
	}
}
