// Package distlock provides distributed locking implementations.
//
// Distributed locks ensure mutual exclusion across multiple processes
// or machines. This is essential for:
//   - Leader election
//   - Preventing duplicate processing
//   - Coordinating access to shared resources
package distlock

import (
	"context"
	"time"
)

// Lock represents a distributed lock.
type Lock interface {
	// Acquire attempts to acquire the lock.
	// Returns true if the lock was acquired, false if already held by another process.
	Acquire(ctx context.Context) (bool, error)

	// Release releases the lock.
	// Should only be called by the holder.
	Release(ctx context.Context) error

	// Extend extends the lock's TTL.
	// Used to prevent expiration during long operations.
	Extend(ctx context.Context, ttl time.Duration) error

	// IsHeld returns true if this lock instance holds the lock.
	IsHeld() bool
}

// Locker creates locks for a given resource.
type Locker interface {
	// NewLock creates a new lock for the given resource key.
	NewLock(key string, ttl time.Duration) Lock

	// Close releases any resources held by the locker.
	Close() error
}

// LockConfig configures lock behavior.
type LockConfig struct {
	// TTL is how long the lock is held before automatic release.
	TTL time.Duration

	// RetryDelay is the delay between acquisition attempts.
	RetryDelay time.Duration

	// RetryCount is the maximum number of acquisition attempts.
	RetryCount int
}

// DefaultLockConfig returns sensible defaults.
func DefaultLockConfig() LockConfig {
	return LockConfig{
		TTL:        10 * time.Second,
		RetryDelay: 50 * time.Millisecond,
		RetryCount: 100,
	}
}

// AcquireWithRetry attempts to acquire lock using cfg.RetryCount and cfg.RetryDelay.
// Returns true if acquired. Honors ctx cancellation between attempts.
// A RetryCount of 0 means a single attempt (no retries after the first failure to acquire).
func AcquireWithRetry(ctx context.Context, lock Lock, cfg LockConfig) (bool, error) {
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = DefaultLockConfig().RetryDelay
	}
	if cfg.RetryCount < 0 {
		cfg.RetryCount = 0
	}

	attempts := cfg.RetryCount + 1
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		acquired, err := lock.Acquire(ctx)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}

		if i == attempts-1 {
			break
		}

		timer := time.NewTimer(cfg.RetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, ctx.Err()
		case <-timer.C:
		}
	}
	return false, nil
}
