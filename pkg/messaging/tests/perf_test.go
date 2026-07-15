package messaging_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
)

func BenchmarkMemoryBroker_Publish(b *testing.B) {
	broker := memory.New(memory.Config{BufferSize: 10000})
	defer broker.Close()

	producer, _ := broker.Producer("benchmark")
	ctx := context.Background()

	msg := &messaging.Message{
		ID:      "bench-msg",
		Payload: []byte(`{"test": "data", "value": 12345}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = producer.Publish(ctx, msg)
	}
}

func BenchmarkMemoryBroker_PublishWithConsumer(b *testing.B) {
	broker := memory.New(memory.Config{BufferSize: 10000})
	defer broker.Close()

	producer, _ := broker.Producer("benchmark")
	consumer, _ := broker.Consumer("benchmark", "bench-group")
	ctx := context.Background()

	// Start consumer
	go func() {
		_ = consumer.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			return nil
		})
	}()

	msg := &messaging.Message{
		ID:      "bench-msg",
		Payload: []byte(`{"test": "data"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = producer.Publish(ctx, msg)
	}
}

func BenchmarkMemoryBroker_Parallel(b *testing.B) {
	broker := memory.New(memory.Config{BufferSize: 100000})
	defer broker.Close()

	producer, _ := broker.Producer("parallel")
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			msg := &messaging.Message{
				ID:      fmt.Sprintf("msg-%d", i),
				Payload: []byte(`{"parallel": true}`),
			}
			_ = producer.Publish(ctx, msg)
			i++
		}
	})
}
