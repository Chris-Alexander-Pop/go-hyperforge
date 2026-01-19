package delay

import (
	"testing"
	"time"
)

func TestDelayQueue(t *testing.T) {
	q := New[string]()

	t.Run("EnqueueDequeue", func(t *testing.T) {
		q.Enqueue("fast", 10*time.Millisecond)
		q.Enqueue("slow", 100*time.Millisecond)

		start := time.Now()

		val, ok := q.Dequeue()
		if !ok || val != "fast" {
			t.Errorf("Expected 'fast', got %v", val)
		}
		if time.Since(start) < 10*time.Millisecond {
			t.Error("Dequeued too early")
		}

		val, ok = q.Dequeue()
		if !ok || val != "slow" {
			t.Errorf("Expected 'slow', got %v", val)
		}
	})

	t.Run("Len", func(t *testing.T) {
		q2 := New[int]()
		q2.Enqueue(1, time.Hour)
		q2.Enqueue(2, time.Hour)
		if q2.Len() != 2 {
			t.Errorf("Expected Len 2, got %d", q2.Len())
		}
	})

	t.Run("Close", func(t *testing.T) {
		q3 := New[int]()
		go func() {
			time.Sleep(50 * time.Millisecond)
			q3.Close()
		}()

		_, ok := q3.Dequeue()
		if ok {
			t.Error("Expected false from Dequeue on closed queue")
		}
	})
}
