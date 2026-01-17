package messaging

// Config holds the base configuration for messaging.
// Each adapter has its own detailed configuration struct.
type Config struct {
	// Driver specifies which messaging adapter to use.
	// Supported values: memory, kafka, rabbitmq, nats, sqs, sns, gcppubsub, azservicebus
	Driver string `env:"MESSAGING_DRIVER" env-default:"memory"`
}
