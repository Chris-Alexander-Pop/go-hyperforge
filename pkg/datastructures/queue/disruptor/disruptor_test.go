package disruptor

import (
	"sync"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := New[int](16)

	t.Run("PublishConsume", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				val := i
				rb.Publish(func(slot *int) {
					*slot = val
				})
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				rb.Consume(func(val int) {
					if val != i {
						t.Errorf("Expected %d, got %d", i, val)
					}
				})
			}
		}()

		wg.Wait()
	})

	t.Run("DefaultSize", func(t *testing.T) {
		rbDefault := New[int](0)
		if len(rbDefault.buffer) != 1024 {
			t.Errorf("Expected default size 1024, got %d", len(rbDefault.buffer))
		}

		rbOdd := New[int](15)
		if len(rbOdd.buffer) != 1024 {
			t.Errorf("Expected size correction to 1024 for non-power-of-2, got %d", len(rbOdd.buffer))
		}
	})
}
