package disruptor_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/queue/disruptor"
	"sync"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := disruptor.New[int](16)

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
		rbDefault := disruptor.New[int](0)
		if len(rbDefault.Buffer) != 1024 {
			t.Errorf("Expected default size 1024, got %d", len(rbDefault.Buffer))
		}

		rbOdd := disruptor.New[int](15)
		if len(rbOdd.Buffer) != 1024 {
			t.Errorf("Expected size correction to 1024 for non-power-of-2, got %d", len(rbOdd.Buffer))
		}
	})
}
