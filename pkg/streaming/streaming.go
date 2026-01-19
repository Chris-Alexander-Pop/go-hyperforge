package streaming

import "context"

// Config holds configuration for streaming clients.
type Config struct {
	// Provider specifies the backend: "memory", "kinesis", "pubsub", "eventhubs".
	Provider string `env:"STREAMING_PROVIDER" env-default:"memory"`

	// Region is the cloud region.
	Region string `env:"STREAMING_REGION"`

	// BufferSize for batching (optional).
	BufferSize int `env:"STREAMING_BUFFER_SIZE" env-default:"100"`
}

// Client abstracts real-time data streaming services.
type Client interface {
	// PutRecord writes a single data record to a stream.
	PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error

	// Close closes the client and flushes any buffers.
	Close() error
}
