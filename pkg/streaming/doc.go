/*
Package streaming provides a unified interface for real-time data streaming backends.

Supported backends:
  - Memory: For local development and testing.
  - AWS Kinesis
  - GCP Pub/Sub
  - Azure Event Hubs

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/streaming"
	import "github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/memory"

	// Create client
	client := memory.New(streaming.Config{BufferSize: 100})

	// Send data
	err := client.PutRecord(ctx, "orders", "user-123", []byte("data"))
*/
package streaming
