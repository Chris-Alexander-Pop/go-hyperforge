package concurrency

import (
	"context"
	"sync"
)

type Semaphore struct {
	size    int64
	cur     int64
	mu      sync.Mutex
	waiters []*waiter
}

type waiter struct {
	n     int64
	ready chan struct{}
}

func NewSemaphore(limit int64) *Semaphore {
	return &Semaphore{
		size: limit,
	}
}

func (s *Semaphore) Acquire(ctx context.Context, n int64) error {
	s.mu.Lock()
	if s.size-s.cur >= n && len(s.waiters) == 0 {
		s.cur += n
		s.mu.Unlock()
		return nil
	}

	w := &waiter{n: n, ready: make(chan struct{})}
	s.waiters = append(s.waiters, w)
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		// Cleanup waiter
		s.mu.Lock()
		// O(N) cleanup but N is presumably small wait queue
		for i, waiter := range s.waiters {
			if waiter == w {
				s.waiters = append(s.waiters[:i], s.waiters[i+1:]...)
				break
			}
		}
		// If we were notified just as we cancelled:
		select {
		case <-w.ready:
			// We got the lock but cancelled. Release it.
			s.cur -= n
			s.notifyWaiters()
		default:
		}
		s.mu.Unlock()
		return ctx.Err()
	case <-w.ready:
		return nil
	}
}

func (s *Semaphore) TryAcquire(n int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.size-s.cur >= n && len(s.waiters) == 0 {
		s.cur += n
		return true
	}
	return false
}

func (s *Semaphore) Release(n int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cur -= n
	if s.cur < 0 {
		// Programming error. Panic is appropriate for sync primitives misuse.
		panic("semaphore released more than held")
	}
	s.notifyWaiters()
}

func (s *Semaphore) notifyWaiters() {
	for {
		if len(s.waiters) == 0 {
			break
		}
		w := s.waiters[0]
		if s.size-s.cur >= w.n {
			s.cur += w.n
			s.waiters = s.waiters[1:]
			close(w.ready)
		} else {
			break
		}
	}
}
