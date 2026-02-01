package delay_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/queue/delay"
)

func BenchmarkSpuriousWakeups(b *testing.B) {
	q := delay.New[int]()
	defer q.Close()

	// 1. Enqueue a head item with a long delay
	q.Enqueue(0, 10*time.Second)

	// 2. Start a consumer that just waits
	done := make(chan struct{})
	go func() {
		defer close(done)
		// We expect this to NOT return until we close the queue or time passes
		q.Dequeue()
	}()

	// Ensure consumer is running and waiting
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Enqueue items that are LATER than the head.
		// These should ideally NOT wake up the consumer.
		q.Enqueue(i+1, 20*time.Second)
	}
	b.StopTimer()
}
