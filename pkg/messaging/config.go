package messaging

// Config holds the base configuration for messaging.
// Each adapter has its own detailed configuration struct for TLS, prefetch,
// credentials, and broker endpoints — those fields are intentionally not
// duplicated here so NewFromConfig stays SDK-light.
type Config struct {
	// Driver specifies which messaging adapter to use.
	// Supported values: memory, kafka, rabbitmq, nats, sqs, sns, gcppubsub,
	// azservicebus, redisstreams.
	//
	// NewFromConfig only constructs registered drivers (memory after importing
	// adapters/memory). Other drivers: use the adapter's New constructor directly.
	Driver string `env:"MESSAGING_DRIVER" env-default:"memory"`

	// BufferSize is the per-consumer channel capacity for the memory driver.
	// Honored by adapters/memory; ignored by network brokers.
	BufferSize int `env:"MESSAGING_BUFFER_SIZE" env-default:"1000"`
}
