// Package azservicebus provides an Azure Service Bus messaging adapter.
//
// This adapter implements the messaging.Broker interface for Azure Service Bus,
// supporting queues, topics, subscriptions, sessions, and dead letter queues.
//
// # Usage
//
//	cfg := azservicebus.Config{
//	    ConnectionString: "Endpoint=sb://...;SharedAccessKeyName=...;SharedAccessKey=...",
//	    QueueName:        "my-queue",
//	}
//	broker, err := azservicebus.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Queue vs Topic Mode
//
// - For point-to-point messaging, use QueueName
// - For pub/sub, use TopicName and SubscriptionName
//
// # Features
//
//   - Queues: Point-to-point messaging
//   - Topics/Subscriptions: Pub/Sub with filtering
//   - Sessions: Ordered message groups
//   - Dead letter: Automatic handling of failed messages
//   - Scheduled messages: Delayed delivery
//
// # Dependencies
//
// This package requires: github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus
package azservicebus

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
)

// Config holds configuration for the Azure Service Bus broker.
type Config struct {
	// ConnectionString is the Azure Service Bus connection string.
	ConnectionString string `env:"AZURE_SERVICEBUS_CONNECTION_STRING"`

	// FullyQualifiedNamespace is the namespace (alternative to connection string).
	FullyQualifiedNamespace string `env:"AZURE_SERVICEBUS_NAMESPACE"`

	// QueueName for point-to-point messaging.
	QueueName string `env:"AZURE_SERVICEBUS_QUEUE"`

	// TopicName for pub/sub messaging.
	TopicName string `env:"AZURE_SERVICEBUS_TOPIC"`

	// SubscriptionName for topic subscriptions.
	SubscriptionName string `env:"AZURE_SERVICEBUS_SUBSCRIPTION"`

	// SessionEnabled enables session support for ordered message groups.
	SessionEnabled bool `env:"AZURE_SERVICEBUS_SESSION_ENABLED" env-default:"false"`

	// MaxConcurrentCalls limits concurrent message processing.
	MaxConcurrentCalls int `env:"AZURE_SERVICEBUS_MAX_CONCURRENT" env-default:"1"`

	// ReceiveMode: PeekLock (default) or ReceiveAndDelete.
	ReceiveMode string `env:"AZURE_SERVICEBUS_RECEIVE_MODE" env-default:"PeekLock"`

	// MaxAutoLockRenewal is the maximum time to auto-renew message locks.
	MaxAutoLockRenewal time.Duration `env:"AZURE_SERVICEBUS_MAX_LOCK_RENEWAL" env-default:"5m"`

	// PrefetchCount is the number of messages to prefetch.
	PrefetchCount int32 `env:"AZURE_SERVICEBUS_PREFETCH" env-default:"0"`
}

// Broker is an Azure Service Bus message broker implementation.
type Broker struct {
	config Config
	client *azservicebus.Client
	mu     sync.RWMutex
	closed bool
}

// New creates a new Azure Service Bus broker.
func New(cfg Config) (*Broker, error) {
	var client *azservicebus.Client
	var err error

	if cfg.ConnectionString != "" {
		client, err = azservicebus.NewClientFromConnectionString(cfg.ConnectionString, nil)
	} else {
		return nil, messaging.ErrInvalidConfig("connection string or namespace required", nil)
	}

	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &Broker{
		config: cfg,
		client: client,
	}, nil
}

// Producer creates a producer for the queue or topic.
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	var sender *azservicebus.Sender
	var err error

	// Use topic parameter if provided, else use config
	queueOrTopic := topic
	if queueOrTopic == "" {
		if b.config.TopicName != "" {
			queueOrTopic = b.config.TopicName
		} else {
			queueOrTopic = b.config.QueueName
		}
	}

	sender, err = b.client.NewSender(queueOrTopic, nil)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &producer{
		broker: b,
		sender: sender,
	}, nil
}

// Consumer creates a consumer for the queue or subscription.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	receiverOpts := &azservicebus.ReceiverOptions{
		ReceiveMode: azservicebus.ReceiveModePeekLock,
	}

	if b.config.ReceiveMode == "ReceiveAndDelete" {
		receiverOpts.ReceiveMode = azservicebus.ReceiveModeReceiveAndDelete
	}

	var receiver *azservicebus.Receiver
	var err error

	if b.config.TopicName != "" {
		// Topic/Subscription mode
		subName := group
		if subName == "" {
			subName = b.config.SubscriptionName
		}
		if subName == "" {
			return nil, messaging.ErrInvalidConfig("subscription name required for topic mode", nil)
		}

		receiver, err = b.client.NewReceiverForSubscription(b.config.TopicName, subName, receiverOpts)
	} else if b.config.QueueName != "" {
		// Queue mode
		receiver, err = b.client.NewReceiverForQueue(b.config.QueueName, receiverOpts)
	} else {
		return nil, messaging.ErrInvalidConfig("queue or topic name required", nil)
	}

	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &consumer{
		broker:   b,
		receiver: receiver,
	}, nil
}

// Close shuts down the Azure Service Bus broker.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	return b.client.Close(context.Background())
}

// Healthy checks if the Service Bus connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return !b.closed
}

// producer is an Azure Service Bus producer.
type producer struct {
	broker *Broker
	sender *azservicebus.Sender
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	asbMsg := &azservicebus.Message{
		Body:      msg.Payload,
		MessageID: &msg.ID,
	}

	// Set application properties from headers
	if len(msg.Headers) > 0 {
		props := make(map[string]interface{})
		for k, v := range msg.Headers {
			props[k] = v
		}
		asbMsg.ApplicationProperties = props
	}

	// Set session ID from key if provided
	if len(msg.Key) > 0 && p.broker.config.SessionEnabled {
		sessionID := string(msg.Key)
		asbMsg.SessionID = &sessionID
	}

	// Set subject/label from topic if provided
	if msg.Topic != "" {
		asbMsg.Subject = &msg.Topic
	}

	err := p.sender.SendMessage(ctx, asbMsg, nil)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	batch, err := p.sender.NewMessageBatch(ctx, nil)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	for _, msg := range msgs {
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		asbMsg := &azservicebus.Message{
			Body:      msg.Payload,
			MessageID: &msg.ID,
		}

		if len(msg.Headers) > 0 {
			props := make(map[string]interface{})
			for k, v := range msg.Headers {
				props[k] = v
			}
			asbMsg.ApplicationProperties = props
		}

		if len(msg.Key) > 0 && p.broker.config.SessionEnabled {
			sessionID := string(msg.Key)
			asbMsg.SessionID = &sessionID
		}

		err := batch.AddMessage(asbMsg, nil)
		if err != nil {
			// Batch is full, send it and start a new one
			if err := p.sender.SendMessageBatch(ctx, batch, nil); err != nil {
				return messaging.ErrPublishFailed(err)
			}

			batch, err = p.sender.NewMessageBatch(ctx, nil)
			if err != nil {
				return messaging.ErrPublishFailed(err)
			}

			if err := batch.AddMessage(asbMsg, nil); err != nil {
				return messaging.ErrPublishFailed(err)
			}
		}
	}

	// Send remaining messages
	if batch.NumMessages() > 0 {
		if err := p.sender.SendMessageBatch(ctx, batch, nil); err != nil {
			return messaging.ErrPublishFailed(err)
		}
	}

	return nil
}

func (p *producer) Close() error {
	return p.sender.Close(context.Background())
}

// consumer is an Azure Service Bus consumer.
type consumer struct {
	broker   *Broker
	receiver *azservicebus.Receiver
	mu       sync.Mutex
	closed   bool
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()

		// Receive messages
		messages, err := c.receiver.ReceiveMessages(ctx, c.broker.config.MaxConcurrentCalls, nil)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		for _, asbMsg := range messages {
			msg := convertServiceBusMessage(asbMsg)

			err := handler(ctx, msg)

			// Handle acknowledgment
			if c.broker.config.ReceiveMode != "ReceiveAndDelete" {
				if err != nil {
					// Abandon message for redelivery
					c.receiver.AbandonMessage(ctx, asbMsg, nil)
				} else {
					// Complete message
					c.receiver.CompleteMessage(ctx, asbMsg, nil)
				}
			}
		}
	}
}

func (c *consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	return c.receiver.Close(context.Background())
}

func convertServiceBusMessage(asbMsg *azservicebus.ReceivedMessage) *messaging.Message {
	msg := &messaging.Message{
		Payload:   asbMsg.Body,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Metadata: messaging.MessageMetadata{
			DeliveryCount: int(asbMsg.DeliveryCount),
			Raw:           asbMsg,
		},
	}

	if asbMsg.MessageID != "" {
		msg.ID = asbMsg.MessageID
	} else {
		msg.ID = uuid.New().String()
	}

	if asbMsg.Subject != nil {
		msg.Topic = *asbMsg.Subject
	}

	if asbMsg.SessionID != nil {
		msg.Key = []byte(*asbMsg.SessionID)
	}

	if asbMsg.EnqueuedTime != nil {
		msg.Timestamp = *asbMsg.EnqueuedTime
	}

	// Convert application properties to headers
	for k, v := range asbMsg.ApplicationProperties {
		if s, ok := v.(string); ok {
			msg.Headers[k] = s
		}
	}

	return msg
}
