package streaming

import "context"

// Config holds configuration for streaming clients.
type Config struct {
	// Provider specifies the backend: "memory", "kinesis", "eventhubs".
	// GCP Pub/Sub and Kafka belong under pkg/messaging, not streaming.
	Provider string `env:"STREAMING_PROVIDER" env-default:"memory"`

	// Region is the cloud region (used by cloud adapters).
	Region string `env:"STREAMING_REGION"`

	// BufferSize caps how many records the memory adapter retains.
	// When the buffer is full, PutRecord returns ErrBufferFull.
	// A value <= 0 means unlimited (no capacity cap).
	BufferSize int `env:"STREAMING_BUFFER_SIZE" env-default:"100"`
}

// Client abstracts real-time data streaming services (append-only record producers).
//
// For broker-style publish/subscribe (Kafka, GCP Pub/Sub, NATS, SQS, …), use pkg/messaging.
type Client interface {
	// PutRecord writes a single data record to a stream.
	PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error

	// Close closes the client and flushes any buffers.
	Close() error
}
