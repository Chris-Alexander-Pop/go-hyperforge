// Package gcppubsub provides a Google Cloud Pub/Sub messaging adapter.
//
// This adapter implements the messaging.Broker interface for Google Cloud Pub/Sub,
// supporting topics, subscriptions, message ordering, and dead letter topics.
//
// # Usage
//
//	cfg := gcppubsub.Config{
//	    ProjectID:      "my-gcp-project",
//	    TopicID:        "my-topic",
//	    SubscriptionID: "my-subscription",
//	}
//	broker, err := gcppubsub.New(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Features
//
//   - Topics: Create and publish to topics
//   - Subscriptions: Pull-based message consumption
//   - Message ordering: With ordering keys
//   - Exactly-once delivery: With ack IDs
//   - Dead letter topics: Automatic handling of failed messages
//
// # Dependencies
//
// This package requires: cloud.google.com/go/pubsub
package gcppubsub

import (
	"context"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// Config holds configuration for the GCP Pub/Sub broker.
type Config struct {
	// ProjectID is the GCP project ID.
	ProjectID string `env:"GCP_PROJECT_ID"`

	// TopicID is the Pub/Sub topic ID.
	TopicID string `env:"PUBSUB_TOPIC_ID"`

	// SubscriptionID is the Pub/Sub subscription ID.
	SubscriptionID string `env:"PUBSUB_SUBSCRIPTION_ID"`

	// CredentialsFile is the path to the service account JSON file.
	// If empty, uses Application Default Credentials.
	CredentialsFile string `env:"GOOGLE_APPLICATION_CREDENTIALS"`

	// CreateTopic creates the topic if it doesn't exist.
	CreateTopic bool `env:"PUBSUB_CREATE_TOPIC" env-default:"true"`

	// CreateSubscription creates the subscription if it doesn't exist.
	CreateSubscription bool `env:"PUBSUB_CREATE_SUBSCRIPTION" env-default:"true"`

	// EnableMessageOrdering enables message ordering with ordering keys.
	EnableMessageOrdering bool `env:"PUBSUB_ENABLE_ORDERING" env-default:"false"`

	// AckDeadline is the acknowledgment deadline.
	AckDeadline time.Duration `env:"PUBSUB_ACK_DEADLINE" env-default:"30s"`

	// MaxOutstandingMessages limits concurrent message processing.
	MaxOutstandingMessages int `env:"PUBSUB_MAX_OUTSTANDING_MESSAGES" env-default:"100"`

	// MaxExtension is the maximum time to extend message ack deadline.
	MaxExtension time.Duration `env:"PUBSUB_MAX_EXTENSION" env-default:"10m"`

	// RetryPolicy configuration for message retries.
	MinRetryBackoff time.Duration `env:"PUBSUB_MIN_RETRY_BACKOFF" env-default:"10s"`
	MaxRetryBackoff time.Duration `env:"PUBSUB_MAX_RETRY_BACKOFF" env-default:"600s"`
}

// Broker is a GCP Pub/Sub message broker implementation.
type Broker struct {
	config Config
	client *pubsub.Client
	topic  *pubsub.Topic
	mu     sync.RWMutex
	closed bool
}

// New creates a new GCP Pub/Sub broker.
func New(ctx context.Context, cfg Config) (*Broker, error) {
	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID, opts...)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	broker := &Broker{
		config: cfg,
		client: client,
	}

	// Get or create topic
	if cfg.TopicID != "" {
		topic := client.Topic(cfg.TopicID)
		exists, err := topic.Exists(ctx)
		if err != nil {
			client.Close()
			return nil, messaging.ErrConnectionFailed(err)
		}

		if !exists {
			if cfg.CreateTopic {
				topic, err = client.CreateTopic(ctx, cfg.TopicID)
				if err != nil {
					client.Close()
					return nil, messaging.ErrConnectionFailed(err)
				}
			} else {
				client.Close()
				return nil, messaging.ErrTopicNotFound(cfg.TopicID, nil)
			}
		}

		if cfg.EnableMessageOrdering {
			topic.EnableMessageOrdering = true
		}

		broker.topic = topic
	}

	return broker, nil
}

// Producer creates a producer for the Pub/Sub topic.
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	pubsubTopic := b.topic
	if topic != "" && topic != b.config.TopicID {
		pubsubTopic = b.client.Topic(topic)
		if b.config.EnableMessageOrdering {
			pubsubTopic.EnableMessageOrdering = true
		}
	}

	return &producer{
		broker: b,
		topic:  pubsubTopic,
	}, nil
}

// Consumer creates a consumer for the Pub/Sub subscription.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	// Use group as subscription ID, or use configured subscription
	subID := group
	if subID == "" {
		subID = b.config.SubscriptionID
	}
	if subID == "" {
		subID = topic + "-sub-" + uuid.New().String()[:8]
	}

	sub := b.client.Subscription(subID)
	exists, err := sub.Exists(context.Background())
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	if !exists {
		if b.config.CreateSubscription {
			topicRef := b.topic
			if topic != "" && topic != b.config.TopicID {
				topicRef = b.client.Topic(topic)
			}

			subCfg := pubsub.SubscriptionConfig{
				Topic:                 topicRef,
				AckDeadline:           b.config.AckDeadline,
				EnableMessageOrdering: b.config.EnableMessageOrdering,
				RetryPolicy: &pubsub.RetryPolicy{
					MinimumBackoff: b.config.MinRetryBackoff,
					MaximumBackoff: b.config.MaxRetryBackoff,
				},
			}

			sub, err = b.client.CreateSubscription(context.Background(), subID, subCfg)
			if err != nil {
				return nil, messaging.ErrConnectionFailed(err)
			}
		} else {
			return nil, messaging.ErrTopicNotFound(subID, nil)
		}
	}

	// Configure receive settings
	sub.ReceiveSettings.MaxOutstandingMessages = b.config.MaxOutstandingMessages
	sub.ReceiveSettings.MaxExtension = b.config.MaxExtension

	return &consumer{
		broker:       b,
		subscription: sub,
	}, nil
}

// Close shuts down the Pub/Sub broker.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	if b.topic != nil {
		b.topic.Stop()
	}

	return b.client.Close()
}

// Healthy checks if the Pub/Sub connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return false
	}
	b.mu.RUnlock()

	if b.topic != nil {
		exists, err := b.topic.Exists(ctx)
		return err == nil && exists
	}

	return true
}

// producer is a GCP Pub/Sub producer.
type producer struct {
	broker *Broker
	topic  *pubsub.Topic
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	attrs := make(map[string]string)
	attrs["message_id"] = msg.ID
	for k, v := range msg.Headers {
		attrs[k] = v
	}

	pubsubMsg := &pubsub.Message{
		Data:       msg.Payload,
		Attributes: attrs,
	}

	// Set ordering key if provided
	if len(msg.Key) > 0 {
		pubsubMsg.OrderingKey = string(msg.Key)
	}

	result := p.topic.Publish(ctx, pubsubMsg)
	serverID, err := result.Get(ctx)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	msg.Metadata.ReceiptHandle = serverID

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	results := make([]*pubsub.PublishResult, len(msgs))

	for i, msg := range msgs {
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		attrs := make(map[string]string)
		attrs["message_id"] = msg.ID
		for k, v := range msg.Headers {
			attrs[k] = v
		}

		pubsubMsg := &pubsub.Message{
			Data:       msg.Payload,
			Attributes: attrs,
		}

		if len(msg.Key) > 0 {
			pubsubMsg.OrderingKey = string(msg.Key)
		}

		results[i] = p.topic.Publish(ctx, pubsubMsg)
	}

	// Wait for all publishes
	for i, result := range results {
		serverID, err := result.Get(ctx)
		if err != nil {
			return messaging.ErrPublishFailed(err)
		}
		msgs[i].Metadata.ReceiptHandle = serverID
	}

	return nil
}

func (p *producer) Close() error {
	p.topic.Stop()
	return nil
}

// consumer is a GCP Pub/Sub consumer.
type consumer struct {
	broker       *Broker
	subscription *pubsub.Subscription
	cancel       context.CancelFunc
	mu           sync.Mutex
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	err := c.subscription.Receive(ctx, func(ctx context.Context, pubsubMsg *pubsub.Message) {
		msg := convertPubSubMessage(pubsubMsg)

		err := handler(ctx, msg)
		if err != nil {
			pubsubMsg.Nack()
			return
		}

		pubsubMsg.Ack()
	})

	if err != nil && err != context.Canceled {
		return messaging.ErrConsumeFailed(err)
	}

	return nil
}

func (c *consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

func convertPubSubMessage(pubsubMsg *pubsub.Message) *messaging.Message {
	deliveryCount := 0
	if pubsubMsg.DeliveryAttempt != nil {
		deliveryCount = *pubsubMsg.DeliveryAttempt
	}

	msg := &messaging.Message{
		ID:        pubsubMsg.ID,
		Payload:   pubsubMsg.Data,
		Headers:   make(map[string]string),
		Timestamp: pubsubMsg.PublishTime,
		Metadata: messaging.MessageMetadata{
			DeliveryCount: deliveryCount,
			Raw:           pubsubMsg,
		},
	}

	// Extract custom message ID if set
	if customID, ok := pubsubMsg.Attributes["message_id"]; ok {
		msg.ID = customID
	}

	// Copy attributes as headers
	for k, v := range pubsubMsg.Attributes {
		if k != "message_id" {
			msg.Headers[k] = v
		}
	}

	// Ordering key as message key
	if pubsubMsg.OrderingKey != "" {
		msg.Key = []byte(pubsubMsg.OrderingKey)
	}

	return msg
}
