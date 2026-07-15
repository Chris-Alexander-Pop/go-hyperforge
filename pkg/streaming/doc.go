/*
Package streaming provides a unified interface for real-time data streaming backends.

Supported backends:
  - Memory: For local development and testing (honors Config.BufferSize).
  - AWS Kinesis
  - Azure Event Hubs

GCP Pub/Sub and Apache Kafka are messaging brokers — use pkg/messaging
(adapters/gcppubsub and adapters/kafka) instead of this package.

Client supports PutRecord and batch PutRecords. An optional Consumer interface
is implemented by the memory adapter for local drain/testing.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/streaming"
	import "github.com/chris-alexander-pop/go-hyperforge/pkg/streaming/adapters/memory"

	client := memory.New(streaming.Config{BufferSize: 100})
	defer client.Close()

	err := client.PutRecord(ctx, "orders", "user-123", []byte("data"))
	err = client.PutRecords(ctx, []streaming.Record{{
		StreamName: "orders", PartitionKey: "user-123", Data: []byte("data"),
	}})

	consumer := client.NewConsumer()
	_ = consumer.Consume(ctx, "orders", func(ctx context.Context, r streaming.Record) error {
		return nil
	})
*/
package streaming
