// Package rabbitmq provides a RabbitMQ messaging adapter using AMQP 0.9.1.
//
// This adapter implements the messaging.Broker interface for RabbitMQ,
// supporting exchanges, queues, routing keys, and publisher confirms.
//
// # Usage
//
//	cfg := rabbitmq.Config{
//	    URL: "amqp://guest:guest@localhost:5672/",
//	}
//	broker, err := rabbitmq.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Dependencies
//
// This package requires: github.com/rabbitmq/amqp091-go
package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Config holds configuration for the RabbitMQ broker.
type Config struct {
	// URL is the AMQP connection URL.
	URL string `env:"RABBITMQ_URL" env-default:"amqp://guest:guest@localhost:5672/"`

	// ExchangeType is the default exchange type (direct, topic, fanout, headers).
	ExchangeType string `env:"RABBITMQ_EXCHANGE_TYPE" env-default:"topic"`

	// ExchangeName is the default exchange name. If empty, uses topic name as exchange.
	ExchangeName string `env:"RABBITMQ_EXCHANGE_NAME"`

	// Durable makes exchanges and queues durable (survive broker restart).
	Durable bool `env:"RABBITMQ_DURABLE" env-default:"true"`

	// AutoDelete deletes queues when last consumer disconnects.
	AutoDelete bool `env:"RABBITMQ_AUTO_DELETE" env-default:"false"`

	// AutoAck automatically acknowledges messages on delivery.
	AutoAck bool `env:"RABBITMQ_AUTO_ACK" env-default:"false"`

	// PrefetchCount limits unacknowledged messages per consumer.
	PrefetchCount int `env:"RABBITMQ_PREFETCH_COUNT" env-default:"10"`

	// PublisherConfirms enables publisher confirm mode for reliable delivery.
	PublisherConfirms bool `env:"RABBITMQ_PUBLISHER_CONFIRMS" env-default:"true"`

	// ReconnectDelay is the delay between reconnection attempts.
	ReconnectDelay time.Duration `env:"RABBITMQ_RECONNECT_DELAY" env-default:"5s"`
}

// Broker is a RabbitMQ message broker implementation.
type Broker struct {
	config Config
	conn   *amqp.Connection
	mu     sync.RWMutex
	closed bool
}

// New creates a new RabbitMQ broker.
func New(cfg Config) (*Broker, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &Broker{
		config: cfg,
		conn:   conn,
	}, nil
}

// Producer creates a new RabbitMQ producer for the specified topic (exchange).
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	ch, err := b.conn.Channel()
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	// Declare exchange
	exchangeName := topic
	if b.config.ExchangeName != "" {
		exchangeName = b.config.ExchangeName
	}

	err = ch.ExchangeDeclare(
		exchangeName,
		b.config.ExchangeType,
		b.config.Durable,
		b.config.AutoDelete,
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		return nil, messaging.ErrConnectionFailed(err)
	}

	// Enable publisher confirms if configured
	if b.config.PublisherConfirms {
		if err := ch.Confirm(false); err != nil {
			ch.Close()
			return nil, messaging.ErrConnectionFailed(err)
		}
	}

	return &producer{
		broker:       b,
		channel:      ch,
		exchange:     exchangeName,
		routingKey:   topic,
		confirms:     b.config.PublisherConfirms,
		confirmsChan: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

// Consumer creates a new RabbitMQ consumer for the specified topic and queue.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	ch, err := b.conn.Channel()
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	// Set prefetch
	if err := ch.Qos(b.config.PrefetchCount, 0, false); err != nil {
		ch.Close()
		return nil, messaging.ErrConnectionFailed(err)
	}

	// Declare exchange
	exchangeName := topic
	if b.config.ExchangeName != "" {
		exchangeName = b.config.ExchangeName
	}

	err = ch.ExchangeDeclare(
		exchangeName,
		b.config.ExchangeType,
		b.config.Durable,
		b.config.AutoDelete,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, messaging.ErrConnectionFailed(err)
	}

	// Queue name: use group if provided, else generate unique
	queueName := group
	if queueName == "" {
		queueName = topic + "-" + uuid.New().String()[:8]
	}

	// Declare queue
	q, err := ch.QueueDeclare(
		queueName,
		b.config.Durable,
		b.config.AutoDelete,
		group == "", // exclusive if no group (unique consumer)
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, messaging.ErrConnectionFailed(err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,
		topic, // routing key
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &consumer{
		broker:    b,
		channel:   ch,
		queue:     q.Name,
		autoAck:   b.config.AutoAck,
		exclusive: group == "",
	}, nil
}

// Close shuts down the RabbitMQ broker connection.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	return b.conn.Close()
}

// Healthy returns true if the broker connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return false
	}

	return !b.conn.IsClosed()
}

// producer is a RabbitMQ producer implementation.
type producer struct {
	broker       *Broker
	channel      *amqp.Channel
	exchange     string
	routingKey   string
	confirms     bool
	confirmsChan <-chan amqp.Confirmation
	mu           sync.Mutex
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	routingKey := p.routingKey
	if msg.Topic != "" {
		routingKey = msg.Topic
	}

	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}

	publishing := amqp.Publishing{
		ContentType:  "application/octet-stream",
		Body:         msg.Payload,
		MessageId:    msg.ID,
		Timestamp:    msg.Timestamp,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	}

	err := p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		publishing,
	)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	// Wait for confirmation if enabled
	if p.confirms {
		select {
		case confirm := <-p.confirmsChan:
			if !confirm.Ack {
				return messaging.ErrPublishFailed(nil)
			}
		case <-ctx.Done():
			return messaging.ErrTimeout("publish confirm", ctx.Err())
		}
	}

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	for _, msg := range msgs {
		if err := p.Publish(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *producer) Close() error {
	return p.channel.Close()
}

// consumer is a RabbitMQ consumer implementation.
type consumer struct {
	broker    *Broker
	channel   *amqp.Channel
	queue     string
	autoAck   bool
	exclusive bool
	mu        sync.Mutex
	closed    bool
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	deliveries, err := c.channel.Consume(
		c.queue,
		"",        // consumer tag (auto-generated)
		c.autoAck, // auto-ack
		c.exclusive,
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return messaging.ErrConsumeFailed(err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil // Channel closed
			}

			msg := convertAMQPDelivery(d)
			err := handler(ctx, msg)

			if !c.autoAck {
				if err != nil {
					// Nack and requeue on error
					d.Nack(false, true)
				} else {
					d.Ack(false)
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

	return c.channel.Close()
}

func convertAMQPDelivery(d amqp.Delivery) *messaging.Message {
	msg := &messaging.Message{
		ID:        d.MessageId,
		Topic:     d.RoutingKey,
		Payload:   d.Body,
		Timestamp: d.Timestamp,
		Headers:   make(map[string]string),
		Metadata: messaging.MessageMetadata{
			DeliveryCount: int(d.DeliveryTag),
			Raw:           d,
		},
	}

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	for k, v := range d.Headers {
		if s, ok := v.(string); ok {
			msg.Headers[k] = s
		}
	}

	return msg
}
