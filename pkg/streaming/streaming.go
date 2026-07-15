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

// Record is a single streaming payload for batch put/consume APIs.
type Record struct {
	StreamName   string
	PartitionKey string
	Data         []byte
}

// RecordHandler processes a consumed record.
type RecordHandler func(ctx context.Context, record Record) error

// Client abstracts real-time data streaming services (append-only record producers).
//
// For broker-style publish/subscribe (Kafka, GCP Pub/Sub, NATS, SQS, …), use pkg/messaging.
type Client interface {
	// PutRecord writes a single data record to a stream.
	PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error

	// PutRecords writes multiple records. Implementations may batch natively
	// or fall back to sequential PutRecord calls. An empty slice is a no-op.
	PutRecords(ctx context.Context, records []Record) error

	// Close closes the client and flushes any buffers.
	Close() error
}

// Consumer optionally reads records from a stream.
// Not all backends expose consume via this package (Kinesis/Event Hubs SDKs
// remain adapter-specific); the memory adapter implements Consumer.
type Consumer interface {
	// Consume delivers records for streamName to handler until ctx is cancelled
	// or the consumer is closed. Missing streams are a no-op (nil error).
	Consume(ctx context.Context, streamName string, handler RecordHandler) error

	// Close releases consumer resources.
	Close() error
}
