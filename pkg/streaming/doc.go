/*
Package streaming provides a unified interface for real-time data streaming backends.

Supported backends:
  - Memory: For local development and testing (honors Config.BufferSize).
  - AWS Kinesis
  - Azure Event Hubs

GCP Pub/Sub and Apache Kafka are messaging brokers — use pkg/messaging
(adapters/gcppubsub and adapters/kafka) instead of this package.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/streaming"
	import "github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/memory"

	client := memory.New(streaming.Config{BufferSize: 100})
	defer client.Close()

	err := client.PutRecord(ctx, "orders", "user-123", []byte("data"))
*/
package streaming
