package memory

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency/distlock"
)

// Adapter implements distlock.Locker using in-memory storage.
type Adapter struct {
	locks map[string]*lockEntry
	mu    sync.Mutex
}

type lockEntry struct {
	value     string
	expiresAt time.Time
}

func New() *Adapter {
	return &Adapter{
		locks: make(map[string]*lockEntry),
	}
}

func (a *Adapter) NewLock(key string, ttl time.Duration) distlock.Lock {
	return &Lock{
		adapter: a,
		key:     key,
		ttl:     ttl,
		// Value is generated on acquire or set here if we want unique instance identity
		value: time.Now().String(), // Simple unique-ish value for testing
	}
}

func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.locks = make(map[string]*lockEntry)
	return nil
}

// Lock implements an in-memory lock.
type Lock struct {
	adapter *Adapter
	key     string
	value   string
	ttl     time.Duration
	held    bool
}

func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	l.adapter.mu.Lock()
	defer l.adapter.mu.Unlock()

	now := time.Now()

	// Check if lock exists and is not expired
	if entry, exists := l.adapter.locks[l.key]; exists {
		if entry.expiresAt.After(now) {
			return false, nil
		}
		// Lock expired, we can take it
	}

	l.adapter.locks[l.key] = &lockEntry{
		value:     l.value,
		expiresAt: now.Add(l.ttl),
	}
	l.held = true
	return true, nil
}

func (l *Lock) Release(ctx context.Context) error {
	l.adapter.mu.Lock()
	defer l.adapter.mu.Unlock()

	if entry, exists := l.adapter.locks[l.key]; exists {
		if entry.value == l.value {
			delete(l.adapter.locks, l.key)
			l.held = false
		}
	}
	return nil
}

func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	l.adapter.mu.Lock()
	defer l.adapter.mu.Unlock()

	if entry, exists := l.adapter.locks[l.key]; exists {
		if entry.value == l.value {
			entry.expiresAt = time.Now().Add(ttl)
			return nil
		}
	}
	l.held = false
	return nil
}

func (l *Lock) IsHeld() bool {
	return l.held
}
