// Package messaging provides a unified abstraction layer for message brokers.
//
// This package defines the core interfaces for producing and consuming messages
// across different messaging systems (Kafka, RabbitMQ, NATS, AWS SQS/SNS, GCP Pub/Sub,
// Azure Service Bus).
//
// # Architecture
//
// The package follows the adapter pattern with decoupled dependencies:
//   - Core interfaces are defined here (zero external dependencies)
//   - Each adapter lives in its own sub-package (pkg/messaging/adapters/{driver})
//   - Users import only the adapter they need, pulling only that SDK
//
// # Usage
//
//	import (
//	    "github.com/chris-alexander-pop/system-design-library/pkg/messaging"
//	    "github.com/chris-alexander-pop/system-design-library/pkg/messaging/adapters/kafka"
//	)
//
//	// Create a Kafka broker
//	broker, err := kafka.New(kafka.Config{Brokers: []string{"localhost:9092"}})
//
//	// Create a producer
//	producer, err := broker.Producer("my-topic")
//	defer producer.Close()
//
//	// Publish a message
//	err = producer.Publish(ctx, &messaging.Message{
//	    ID:      uuid.New().String(),
//	    Topic:   "my-topic",
//	    Payload: []byte(`{"event": "user.created"}`),
//	})
package messaging

import (
	"context"
	"time"
)

// Message represents a message to be sent or received from a message broker.
// It provides a unified structure across all messaging systems.
type Message struct {
	// ID is a unique identifier for the message.
	// If not provided, adapters should generate one.
	ID string `json:"id"`

	// Topic is the destination topic/queue/subject name.
	Topic string `json:"topic"`

	// Key is used for partitioning in systems that support it (e.g., Kafka).
	// Messages with the same key are guaranteed to go to the same partition.
	Key []byte `json:"key,omitempty"`

	// Payload is the message body.
	Payload []byte `json:"payload"`

	// Headers are optional key-value pairs for metadata.
	Headers map[string]string `json:"headers,omitempty"`

	// Timestamp is when the message was created.
	// If not set, adapters should use the current time.
	Timestamp time.Time `json:"timestamp"`

	// Metadata contains broker-specific information (e.g., partition, offset for Kafka).
	// This is populated by the consumer and should be treated as read-only.
	Metadata MessageMetadata `json:"metadata,omitempty"`
}

// MessageMetadata contains broker-specific information about a message.
type MessageMetadata struct {
	// Partition is the partition number (Kafka, etc.)
	Partition int32 `json:"partition,omitempty"`

	// Offset is the message offset within the partition (Kafka, etc.)
	Offset int64 `json:"offset,omitempty"`

	// DeliveryCount is how many times this message has been delivered (for retry tracking)
	DeliveryCount int `json:"delivery_count,omitempty"`

	// ReceiptHandle is used for acknowledgment in SQS-like systems
	ReceiptHandle string `json:"receipt_handle,omitempty"`

	// Raw contains the original broker-specific message if needed
	Raw interface{} `json:"-"`
}

// MessageHandler processes incoming messages.
// Return nil to acknowledge the message, or an error to trigger retry/nack behavior.
type MessageHandler func(ctx context.Context, msg *Message) error

// Producer sends messages to a topic/queue.
type Producer interface {
	// Publish sends a single message.
	// The message's Topic field is used if set, otherwise the producer's default topic is used.
	Publish(ctx context.Context, msg *Message) error

	// PublishBatch sends multiple messages in a single operation.
	// This is more efficient for high-throughput scenarios.
	PublishBatch(ctx context.Context, msgs []*Message) error

	// Close releases resources associated with the producer.
	Close() error
}

// Consumer receives messages from a topic/queue.
type Consumer interface {
	// Consume starts consuming messages and calls the handler for each one.
	// This method blocks until the context is canceled or an error occurs.
	// The handler's return value controls acknowledgment:
	//   - nil: message is acknowledged
	//   - error: message is not acknowledged (may be redelivered based on broker config)
	Consume(ctx context.Context, handler MessageHandler) error

	// Close stops consuming and releases resources.
	Close() error
}

// Broker manages connections and creates producers/consumers.
// Each adapter implements this interface to provide broker-specific functionality.
type Broker interface {
	// Producer creates a new producer for the specified topic.
	// The producer can be reused for multiple messages.
	Producer(topic string) (Producer, error)

	// Consumer creates a new consumer for the specified topic and consumer group.
	// The group parameter is used for load balancing across multiple consumers.
	// Use an empty string for broadcast/fanout behavior if supported.
	Consumer(topic string, group string) (Consumer, error)

	// Close shuts down the broker connection and all associated producers/consumers.
	Close() error

	// Healthy returns true if the broker connection is healthy.
	Healthy(ctx context.Context) bool
}

// PublishOption configures a publish operation.
type PublishOption func(*publishOptions)

type publishOptions struct {
	// DelaySeconds delays message delivery (SQS, Azure Service Bus)
	DelaySeconds int64
	// OrderingKey ensures messages with the same key are delivered in order (GCP Pub/Sub)
	OrderingKey string
	// MessageGroupID groups messages for FIFO ordering (SQS FIFO)
	MessageGroupID string
	// DeduplicationID prevents duplicate message delivery (SQS FIFO)
	DeduplicationID string
}

// WithDelay sets a delivery delay for the message.
func WithDelay(seconds int64) PublishOption {
	return func(o *publishOptions) {
		o.DelaySeconds = seconds
	}
}

// WithOrderingKey sets the ordering key for message ordering.
func WithOrderingKey(key string) PublishOption {
	return func(o *publishOptions) {
		o.OrderingKey = key
	}
}

// WithMessageGroupID sets the message group for FIFO ordering.
func WithMessageGroupID(groupID string) PublishOption {
	return func(o *publishOptions) {
		o.MessageGroupID = groupID
	}
}

// WithDeduplicationID sets the deduplication ID for exactly-once delivery.
func WithDeduplicationID(dedupID string) PublishOption {
	return func(o *publishOptions) {
		o.DeduplicationID = dedupID
	}
}

// ConsumeOption configures a consume operation.
type ConsumeOption func(*consumeOptions)

type consumeOptions struct {
	// MaxMessages limits the number of messages to fetch at once
	MaxMessages int
	// VisibilityTimeout sets how long a message is hidden after being received
	VisibilityTimeout time.Duration
	// WaitTime sets how long to wait for messages (long polling)
	WaitTime time.Duration
}

// WithMaxMessages sets the maximum number of messages to receive.
func WithMaxMessages(n int) ConsumeOption {
	return func(o *consumeOptions) {
		o.MaxMessages = n
	}
}

// WithVisibilityTimeout sets the visibility timeout for received messages.
func WithVisibilityTimeout(d time.Duration) ConsumeOption {
	return func(o *consumeOptions) {
		o.VisibilityTimeout = d
	}
}

// WithWaitTime sets the wait time for long polling.
func WithWaitTime(d time.Duration) ConsumeOption {
	return func(o *consumeOptions) {
		o.WaitTime = d
	}
}
