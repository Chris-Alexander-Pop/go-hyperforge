package distlock

import (
	"context"
	"sync"
	"time"
)

// MemoryLocker implements in-memory distributed locking for testing.
type MemoryLocker struct {
	locks map[string]*memoryLockEntry
	mu    sync.Mutex
}

type memoryLockEntry struct {
	value     string
	expiresAt time.Time
}

// NewMemoryLocker creates a new in-memory locker for testing.
func NewMemoryLocker() *MemoryLocker {
	return &MemoryLocker{
		locks: make(map[string]*memoryLockEntry),
	}
}

func (l *MemoryLocker) NewLock(key string, ttl time.Duration) Lock {
	return &memoryLock{
		locker: l,
		key:    key,
		value:  time.Now().String(), // Unique value
		ttl:    ttl,
	}
}

func (l *MemoryLocker) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locks = make(map[string]*memoryLockEntry)
	return nil
}

type memoryLock struct {
	locker *MemoryLocker
	key    string
	value  string
	ttl    time.Duration
	held   bool
}

func (l *memoryLock) Acquire(ctx context.Context) (bool, error) {
	l.locker.mu.Lock()
	defer l.locker.mu.Unlock()

	now := time.Now()

	// Check if lock exists and is not expired
	if entry, exists := l.locker.locks[l.key]; exists {
		if entry.expiresAt.After(now) {
			return false, nil
		}
		// Lock expired, we can take it
	}

	l.locker.locks[l.key] = &memoryLockEntry{
		value:     l.value,
		expiresAt: now.Add(l.ttl),
	}
	l.held = true
	return true, nil
}

func (l *memoryLock) Release(ctx context.Context) error {
	l.locker.mu.Lock()
	defer l.locker.mu.Unlock()

	if entry, exists := l.locker.locks[l.key]; exists {
		if entry.value == l.value {
			delete(l.locker.locks, l.key)
			l.held = false
		}
	}
	return nil
}

func (l *memoryLock) Extend(ctx context.Context, ttl time.Duration) error {
	l.locker.mu.Lock()
	defer l.locker.mu.Unlock()

	if entry, exists := l.locker.locks[l.key]; exists {
		if entry.value == l.value {
			entry.expiresAt = time.Now().Add(ttl)
			return nil
		}
	}
	l.held = false
	return nil
}

func (l *memoryLock) IsHeld() bool {
	return l.held
}
