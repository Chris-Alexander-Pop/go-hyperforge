package shaper

import (
	"sync"
	"testing"
	"time"
)

func TestShaper(t *testing.T) {
	// Rate: 10/s, Burst: 1
	s := New(10.0, 1.0)
	defer s.Stop()

	var mu sync.Mutex
	count := 0
	done := make(chan struct{})

	// Task increments count
	task := func() {
		mu.Lock()
		count++
		if count == 3 {
			close(done)
		}
		mu.Unlock()
	}

	// Push 3 tasks rapidly
	s.Push(task)
	s.Push(task)
	s.Push(task)

	// Should complete 1 immediately, then others spaced by 100ms
	select {
	case <-done:
		// check timing if critical, but for now just completion
	case <-time.After(500 * time.Millisecond):
		t.Errorf("Tasks didn't complete in time")
	}

	mu.Lock()
	if count != 3 {
		t.Errorf("Expected 3 tasks, got %d", count)
	}
	mu.Unlock()
}
