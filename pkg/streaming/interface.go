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
