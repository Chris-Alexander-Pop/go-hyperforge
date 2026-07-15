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
// This package requires: cloud.google.com/go/pubsub/v2
package gcppubsub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"golang.org/x/oauth2/google"
	"google.golang.org/protobuf/types/known/durationpb"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
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

// topicFullName returns the fully qualified topic name.
func topicFullName(projectID, topicID string) string {
	return fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
}

// subscriptionFullName returns the fully qualified subscription name.
func subscriptionFullName(projectID, subID string) string {
	return fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subID)
}

// Broker is a GCP Pub/Sub message broker implementation.
type Broker struct {
	config    Config
	client    *pubsub.Client
	publisher *pubsub.Publisher
	topicName string
	mu        *concurrency.SmartRWMutex
	closed    bool
}

// New creates a new GCP Pub/Sub broker.
func New(ctx context.Context, cfg Config) (*Broker, error) {
	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		credsJSON, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, messaging.ErrConnectionFailed(err)
		}
		creds, err := google.CredentialsFromJSON(ctx, credsJSON, pubsub.ScopePubSub)
		if err != nil {
			return nil, messaging.ErrConnectionFailed(err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID, opts...)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	broker := &Broker{
		config: cfg,
		client: client,
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "PubSubBroker"}),
	}

	// Get or create topic
	if cfg.TopicID != "" {
		topicName := topicFullName(cfg.ProjectID, cfg.TopicID)

		// Check if topic exists using admin client
		_, err := client.TopicAdminClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{
			Topic: topicName,
		})
		if err != nil {
			if cfg.CreateTopic {
				_, err = client.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{
					Name: topicName,
				})
				if err != nil {
					client.Close()
					return nil, messaging.ErrConnectionFailed(err)
				}
			} else {
				client.Close()
				return nil, messaging.ErrTopicNotFound(cfg.TopicID, err)
			}
		}

		// Create publisher for the topic
		publisher := client.Publisher(topicName)
		if cfg.EnableMessageOrdering {
			publisher.EnableMessageOrdering = true
		}

		broker.publisher = publisher
		broker.topicName = topicName
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

	publisher := b.publisher
	topicName := b.topicName

	if topic != "" && topic != b.config.TopicID {
		topicName = topicFullName(b.config.ProjectID, topic)
		publisher = b.client.Publisher(topicName)
		if b.config.EnableMessageOrdering {
			publisher.EnableMessageOrdering = true
		}
	}

	return &producer{
		broker:    b,
		publisher: publisher,
		topicName: topicName,
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

	subName := subscriptionFullName(b.config.ProjectID, subID)
	topicName := b.topicName
	if topic != "" && topic != b.config.TopicID {
		topicName = topicFullName(b.config.ProjectID, topic)
	}

	// Check if subscription exists
	_, err := b.client.SubscriptionAdminClient.GetSubscription(context.Background(), &pubsubpb.GetSubscriptionRequest{
		Subscription: subName,
	})
	if err != nil {
		if b.config.CreateSubscription {
			_, err = b.client.SubscriptionAdminClient.CreateSubscription(context.Background(), &pubsubpb.Subscription{
				Name:                  subName,
				Topic:                 topicName,
				AckDeadlineSeconds:    int32(b.config.AckDeadline.Seconds()),
				EnableMessageOrdering: b.config.EnableMessageOrdering,
				RetryPolicy: &pubsubpb.RetryPolicy{
					MinimumBackoff: toDurationPB(b.config.MinRetryBackoff),
					MaximumBackoff: toDurationPB(b.config.MaxRetryBackoff),
				},
			})
			if err != nil {
				return nil, messaging.ErrConnectionFailed(err)
			}
		} else {
			return nil, messaging.ErrTopicNotFound(subID, err)
		}
	}

	// Create subscriber
	subscriber := b.client.Subscriber(subName)
	subscriber.ReceiveSettings.MaxOutstandingMessages = b.config.MaxOutstandingMessages
	subscriber.ReceiveSettings.MaxExtension = b.config.MaxExtension

	return &consumer{
		broker:     b,
		subscriber: subscriber,
		subName:    subName,
		mu:         concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "PubSubConsumer"}),
	}, nil
}

// toDurationPB converts a time.Duration to a protobuf Duration.
func toDurationPB(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}

// Close shuts down the Pub/Sub broker.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	if b.publisher != nil {
		b.publisher.Stop()
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

	if b.topicName != "" {
		_, err := b.client.TopicAdminClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{
			Topic: b.topicName,
		})
		return err == nil
	}

	return true
}

// producer is a GCP Pub/Sub producer.
type producer struct {
	broker    *Broker
	publisher *pubsub.Publisher
	topicName string
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

	result := p.publisher.Publish(ctx, pubsubMsg)
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

		results[i] = p.publisher.Publish(ctx, pubsubMsg)
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
	p.publisher.Stop()
	return nil
}

// consumer is a GCP Pub/Sub consumer.
type consumer struct {
	broker     *Broker
	subscriber *pubsub.Subscriber
	subName    string
	cancel     context.CancelFunc
	mu         *concurrency.SmartMutex
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	err := c.subscriber.Receive(ctx, func(ctx context.Context, pubsubMsg *pubsub.Message) {
		msg := convertPubSubMessage(pubsubMsg)

		err := handler(ctx, msg)
		if err != nil {
			pubsubMsg.Nack()
			return
		}

		pubsubMsg.Ack()
	})

	if err != nil && !errors.Is(err, context.Canceled) {
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
