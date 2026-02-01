package delay_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/queue/delay"
)

func TestDequeueContext(t *testing.T) {
	q := delay.New[string]()

	t.Run("Cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Start Dequeue in background
		errCh := make(chan error)
		go func() {
			_, err := q.DequeueContext(ctx)
			errCh <- err
		}()

		// Allow Dequeue to start waiting
		time.Sleep(50 * time.Millisecond)

		cancel()

		select {
		case err := <-errCh:
			if err != context.Canceled {
				t.Errorf("Expected context.Canceled, got %v", err)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("DequeueContext did not return after cancellation")
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := q.DequeueContext(ctx)
		elapsed := time.Since(start)

		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
		if elapsed < 50*time.Millisecond {
			t.Error("Returned too early")
		}
	})

	t.Run("Success", func(t *testing.T) {
		q.Enqueue("hello", 50*time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		val, err := q.DequeueContext(ctx)
		if err != nil {
			t.Errorf("Expected success, got error %v", err)
		}
		if val != "hello" {
			t.Errorf("Expected 'hello', got %v", val)
		}
	})

	t.Run("Closed", func(t *testing.T) {
		q2 := delay.New[int]()
		q2.Close()
		_, err := q2.DequeueContext(context.Background())
		if err != delay.ErrClosed {
			t.Errorf("Expected ErrClosed, got %v", err)
		}
	})
}
