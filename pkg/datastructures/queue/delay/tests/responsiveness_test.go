package delay_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/queue/delay"
)

func TestOptimizationOpportunity(t *testing.T) {
	q := delay.New[string]()
	defer q.Close()

	// Enqueue a long delay item
	q.Enqueue("long", 2*time.Second)

	// Start a goroutine to Dequeue
	done := make(chan string)
	go func() {
		val, _ := q.Dequeue()
		done <- val
	}()

	// Give Dequeue time to pick up the long item and start sleeping
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	// Enqueue a short delay item.
	// Since 100ms passed, we want this ready in 200ms from NOW (total 300ms from start)
	// which is much earlier than the "long" item (2s from start).
	q.Enqueue("short", 200*time.Millisecond)

	// Expect to get "short" back soon
	select {
	case val := <-done:
		elapsed := time.Since(start)
		if val != "short" {
			t.Errorf("Expected 'short', got '%s'", val)
		}
		// If implementation is efficient, it should return around 200ms.
		// If inefficient (blocking sleep), it will return around 1900ms (remaining of 2s).
		t.Logf("Dequeue took %v", elapsed)
		if elapsed > 1*time.Second {
			t.Errorf("Performance Issue: Dequeue took too long (%v), blocked by previous long item", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for Dequeue")
	}
}
