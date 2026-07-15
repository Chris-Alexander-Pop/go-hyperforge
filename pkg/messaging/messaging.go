package messaging

import (
	"context"
	"strconv"
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

// Well-known header keys used by ApplyPublishOptions so adapters can honor
// broker-specific publish hints without changing the Producer interface.
const (
	HeaderDelaySeconds    = "x-messaging-delay-seconds"
	HeaderOrderingKey     = "x-messaging-ordering-key"
	HeaderMessageGroupID  = "x-messaging-message-group-id"
	HeaderDeduplicationID = "x-messaging-deduplication-id"
)

// PublishOption configures a publish operation.
type PublishOption func(*PublishOptions)

// PublishOptions holds broker hints applied before Publish.
type PublishOptions struct {
	// DelaySeconds delays message delivery (SQS, Azure Service Bus).
	DelaySeconds int64
	// OrderingKey ensures messages with the same key are delivered in order (GCP Pub/Sub).
	OrderingKey string
	// MessageGroupID groups messages for FIFO ordering (SQS FIFO).
	MessageGroupID string
	// DeduplicationID prevents duplicate message delivery (SQS FIFO).
	DeduplicationID string
}

// WithDelay sets a delivery delay for the message.
func WithDelay(seconds int64) PublishOption {
	return func(o *PublishOptions) {
		o.DelaySeconds = seconds
	}
}

// WithOrderingKey sets the ordering key for message ordering.
func WithOrderingKey(key string) PublishOption {
	return func(o *PublishOptions) {
		o.OrderingKey = key
	}
}

// WithMessageGroupID sets the message group for FIFO ordering.
func WithMessageGroupID(groupID string) PublishOption {
	return func(o *PublishOptions) {
		o.MessageGroupID = groupID
	}
}

// WithDeduplicationID sets the deduplication ID for exactly-once delivery.
func WithDeduplicationID(dedupID string) PublishOption {
	return func(o *PublishOptions) {
		o.DeduplicationID = dedupID
	}
}

// ApplyPublishOptions writes publish hints into msg.Headers (and Key when an
// ordering key is set). Adapters may read these via ParsePublishOptions.
func ApplyPublishOptions(msg *Message, opts ...PublishOption) {
	if msg == nil || len(opts) == 0 {
		return
	}
	o := &PublishOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	if o.DelaySeconds > 0 {
		msg.Headers[HeaderDelaySeconds] = formatInt64(o.DelaySeconds)
	}
	if o.OrderingKey != "" {
		msg.Headers[HeaderOrderingKey] = o.OrderingKey
		if len(msg.Key) == 0 {
			msg.Key = []byte(o.OrderingKey)
		}
	}
	if o.MessageGroupID != "" {
		msg.Headers[HeaderMessageGroupID] = o.MessageGroupID
	}
	if o.DeduplicationID != "" {
		msg.Headers[HeaderDeduplicationID] = o.DeduplicationID
	}
}

// ParsePublishOptions reads publish hints previously applied to msg.Headers.
func ParsePublishOptions(msg *Message) PublishOptions {
	if msg == nil || msg.Headers == nil {
		return PublishOptions{}
	}
	return PublishOptions{
		DelaySeconds:    parseInt64(msg.Headers[HeaderDelaySeconds]),
		OrderingKey:     msg.Headers[HeaderOrderingKey],
		MessageGroupID:  msg.Headers[HeaderMessageGroupID],
		DeduplicationID: msg.Headers[HeaderDeduplicationID],
	}
}

// Publish applies opts to msg then delegates to p.Publish.
// Prefer this helper when callers need PublishOption without changing Producer.
func Publish(ctx context.Context, p Producer, msg *Message, opts ...PublishOption) error {
	ApplyPublishOptions(msg, opts...)
	return p.Publish(ctx, msg)
}

// ConsumeOption configures a consume operation.
type ConsumeOption func(*ConsumeOptions)

// ConsumeOptions holds poll/visibility hints for Consume.
type ConsumeOptions struct {
	// MaxMessages limits the number of messages to fetch at once.
	MaxMessages int
	// VisibilityTimeout sets how long a message is hidden after being received.
	VisibilityTimeout time.Duration
	// WaitTime sets how long to wait for messages (long polling).
	WaitTime time.Duration
}

// WithMaxMessages sets the maximum number of messages to receive.
func WithMaxMessages(n int) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.MaxMessages = n
	}
}

// WithVisibilityTimeout sets the visibility timeout for received messages.
func WithVisibilityTimeout(d time.Duration) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.VisibilityTimeout = d
	}
}

// WithWaitTime sets the wait time for long polling.
func WithWaitTime(d time.Duration) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.WaitTime = d
	}
}

type consumeOptionsCtxKey struct{}

// ContextWithConsumeOptions stores consume opts on ctx for adapters to read.
func ContextWithConsumeOptions(ctx context.Context, opts ...ConsumeOption) context.Context {
	if len(opts) == 0 {
		return ctx
	}
	o := &ConsumeOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return context.WithValue(ctx, consumeOptionsCtxKey{}, *o)
}

// ConsumeOptionsFromContext returns consume opts previously stored on ctx.
func ConsumeOptionsFromContext(ctx context.Context) (ConsumeOptions, bool) {
	if ctx == nil {
		return ConsumeOptions{}, false
	}
	o, ok := ctx.Value(consumeOptionsCtxKey{}).(ConsumeOptions)
	return o, ok
}

// Consume stores opts on ctx then delegates to c.Consume.
// SQS/Service Bus adapters may honor MaxMessages / WaitTime / VisibilityTimeout
// from context; the memory adapter ignores them.
func Consume(ctx context.Context, c Consumer, handler MessageHandler, opts ...ConsumeOption) error {
	ctx = ContextWithConsumeOptions(ctx, opts...)
	return c.Consume(ctx, handler)
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
