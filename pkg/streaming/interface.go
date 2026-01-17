// Package streaming provides a unified interface for real-time data streaming.
//
// Supported backends:
//   - AWS Kinesis
//   - GCP Pub/Sub
//   - Azure Event Hubs
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/kinesis"
//
//	client := kinesis.New(cfg)
//	err := client.PutRecord(ctx, "my-stream", partitionKey, data)
package streaming

import (
	"context"
)

// Client abstracts real-time data streaming services.
type Client interface {
	// PutRecord writes a single data record to a stream.
	PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error

	// Close closes the client.
	Close() error
}
