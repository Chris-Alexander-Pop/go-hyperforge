package ring

import (
	"testing"
	"time"
)

func TestRingBuffer(t *testing.T) {
	t.Run("BasicFIFO", func(t *testing.T) {
		rb := New[int](5)
		rb.Enqueue(1)
		rb.Enqueue(2)
		rb.Enqueue(3)

		if rb.Dequeue() != 1 {
			t.Error("Expected 1")
		}
		if rb.Dequeue() != 2 {
			t.Error("Expected 2")
		}
		if rb.Dequeue() != 3 {
			t.Error("Expected 3")
		}
	})

	t.Run("OverflowBlock", func(t *testing.T) {
		rb := New[int](2)
		rb.Enqueue(1)
		rb.Enqueue(2)

		done := make(chan bool)
		go func() {
			rb.Enqueue(3)
			done <- true
		}()

		select {
		case <-done:
			t.Error("Should have blocked on full buffer")
		case <-time.After(50 * time.Millisecond):
			// expected blocking
		}

		rb.Dequeue() // Make space
		<-done       // Should proceed now
	})

	t.Run("UnderflowBlock", func(t *testing.T) {
		rb := New[int](2)
		done := make(chan bool)
		go func() {
			rb.Dequeue()
			done <- true
		}()

		select {
		case <-done:
			t.Error("Should have blocked on empty buffer")
		case <-time.After(50 * time.Millisecond):
			// expected blocking
		}

		rb.Enqueue(1)
		<-done
	})

	t.Run("TryEnqueueDequeue", func(t *testing.T) {
		rb := New[int](1)
		if err := rb.TryEnqueue(1); err != nil {
			t.Error("Expected successful enqueue")
		}

		if err := rb.TryEnqueue(2); err != ErrBufferFull {
			t.Errorf("Expected ErrBufferFull, got %v", err)
		}

		val, err := rb.TryDequeue()
		if err != nil || val != 1 {
			t.Errorf("Expected 1, got %v, err %v", val, err)
		}

		_, err = rb.TryDequeue()
		if err != ErrBufferEmpty {
			t.Errorf("Expected ErrBufferEmpty, got %v", err)
		}
	})

	t.Run("CapAndLen", func(t *testing.T) {
		rb := New[int](10)
		if rb.Cap() != 10 {
			t.Errorf("Expected Cap 10, got %d", rb.Cap())
		}
		rb.Enqueue(1)
		if rb.Len() != 1 {
			t.Errorf("Expected Len 1, got %d", rb.Len())
		}
	})
}
