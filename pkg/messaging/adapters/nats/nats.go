// Package nats provides a NATS messaging adapter with JetStream support.
//
// This adapter implements the messaging.Broker interface for NATS,
// supporting both core NATS pub/sub and JetStream for persistent messaging.
//
// # Usage
//
//	cfg := nats.Config{
//	    URL:             "nats://localhost:4222",
//	    EnableJetStream: true,
//	    StreamName:      "ORDERS",
//	}
//	broker, err := nats.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # JetStream Features
//
// When JetStream is enabled:
//   - Messages are persisted and can be replayed
//   - Consumer groups (durable consumers) are supported
//   - Exactly-once delivery with ack/nack
//   - Work queue pattern with load balancing
//
// # Dependencies
//
// This package requires: github.com/nats-io/nats.go
package nats

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config holds configuration for the NATS broker.
type Config struct {
	// URL is the NATS server URL(s). Multiple URLs can be comma-separated.
	URL string `env:"NATS_URL" env-default:"nats://localhost:4222"`

	// Name is the client connection name.
	Name string `env:"NATS_CLIENT_NAME" env-default:"system-design-library"`

	// EnableJetStream enables JetStream for persistent messaging.
	EnableJetStream bool `env:"NATS_JETSTREAM" env-default:"true"`

	// StreamName is the JetStream stream name.
	StreamName string `env:"NATS_STREAM" env-default:"MESSAGES"`

	// StreamSubjects are the subjects the stream will capture.
	// If empty, defaults to StreamName.>
	StreamSubjects []string `env:"NATS_STREAM_SUBJECTS"`

	// Replicas is the number of stream replicas (for HA).
	Replicas int `env:"NATS_REPLICAS" env-default:"1"`

	// Retention is the retention policy: limits, interest, workqueue.
	Retention string `env:"NATS_RETENTION" env-default:"limits"`

	// MaxMsgs is the maximum number of messages in the stream.
	MaxMsgs int64 `env:"NATS_MAX_MSGS" env-default:"-1"`

	// MaxBytes is the maximum bytes in the stream.
	MaxBytes int64 `env:"NATS_MAX_BYTES" env-default:"-1"`

	// MaxAge is the maximum age of messages.
	MaxAge time.Duration `env:"NATS_MAX_AGE" env-default:"0"`

	// AckWait is the time to wait for ack before redelivery.
	AckWait time.Duration `env:"NATS_ACK_WAIT" env-default:"30s"`

	// MaxDeliver is the maximum delivery attempts.
	MaxDeliver int `env:"NATS_MAX_DELIVER" env-default:"5"`

	// Credentials file path for authentication.
	CredsFile string `env:"NATS_CREDS_FILE"`

	// Token for token authentication.
	Token string `env:"NATS_TOKEN"`

	// User and password for basic auth.
	User     string `env:"NATS_USER"`
	Password string `env:"NATS_PASSWORD"`
}

// Broker is a NATS message broker implementation.
type Broker struct {
	config Config
	conn   *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	mu     sync.RWMutex
	closed bool
}

// New creates a new NATS broker.
func New(cfg Config) (*Broker, error) {
	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
	}

	// Authentication
	if cfg.CredsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.CredsFile))
	} else if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	} else if cfg.User != "" {
		opts = append(opts, nats.UserInfo(cfg.User, cfg.Password))
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	broker := &Broker{
		config: cfg,
		conn:   conn,
	}

	// Initialize JetStream if enabled
	if cfg.EnableJetStream {
		js, err := jetstream.New(conn)
		if err != nil {
			conn.Close()
			return nil, messaging.ErrConnectionFailed(err)
		}
		broker.js = js

		// Create or update stream
		stream, err := broker.ensureStream(context.Background())
		if err != nil {
			conn.Close()
			return nil, err
		}
		broker.stream = stream
	}

	return broker, nil
}

func (b *Broker) ensureStream(ctx context.Context) (jetstream.Stream, error) {
	subjects := b.config.StreamSubjects
	if len(subjects) == 0 {
		subjects = []string{b.config.StreamName + ".>"}
	}

	retention := jetstream.LimitsPolicy
	switch strings.ToLower(b.config.Retention) {
	case "interest":
		retention = jetstream.InterestPolicy
	case "workqueue":
		retention = jetstream.WorkQueuePolicy
	}

	streamCfg := jetstream.StreamConfig{
		Name:      b.config.StreamName,
		Subjects:  subjects,
		Retention: retention,
		Replicas:  b.config.Replicas,
		MaxMsgs:   b.config.MaxMsgs,
		MaxBytes:  b.config.MaxBytes,
		MaxAge:    b.config.MaxAge,
	}

	stream, err := b.js.CreateOrUpdateStream(ctx, streamCfg)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return stream, nil
}

// Producer creates a new NATS producer for the specified subject.
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	if b.config.EnableJetStream {
		return &jetStreamProducer{
			broker:  b,
			subject: b.config.StreamName + "." + topic,
		}, nil
	}

	return &coreProducer{
		broker:  b,
		subject: topic,
	}, nil
}

// Consumer creates a new NATS consumer for the specified subject.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	if b.config.EnableJetStream {
		return b.createJetStreamConsumer(topic, group)
	}

	return b.createCoreConsumer(topic, group)
}

func (b *Broker) createJetStreamConsumer(topic string, group string) (messaging.Consumer, error) {
	subject := b.config.StreamName + "." + topic

	consumerCfg := jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       b.config.AckWait,
		MaxDeliver:    b.config.MaxDeliver,
		FilterSubject: subject,
	}

	if group != "" {
		// Durable consumer with consumer group
		consumerCfg.Durable = group
		consumerCfg.DeliverPolicy = jetstream.DeliverAllPolicy
	} else {
		// Ephemeral consumer - start from now
		consumerCfg.DeliverPolicy = jetstream.DeliverNewPolicy
	}

	consumer, err := b.stream.CreateOrUpdateConsumer(context.Background(), consumerCfg)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &jetStreamConsumer{
		broker:   b,
		consumer: consumer,
		subject:  subject,
		group:    group,
	}, nil
}

func (b *Broker) createCoreConsumer(topic string, group string) (messaging.Consumer, error) {
	return &coreConsumer{
		broker:  b,
		subject: topic,
		group:   group,
	}, nil
}

// Close shuts down the NATS connection.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	b.conn.Close()
	return nil
}

// Healthy returns true if the NATS connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return false
	}

	return b.conn.IsConnected()
}

// coreProducer is a core NATS producer (no persistence).
type coreProducer struct {
	broker  *Broker
	subject string
}

func (p *coreProducer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	natsMsg := &nats.Msg{
		Subject: p.subject,
		Data:    msg.Payload,
		Header:  nats.Header{},
	}

	natsMsg.Header.Set("Nats-Msg-Id", msg.ID)
	for k, v := range msg.Headers {
		natsMsg.Header.Set(k, v)
	}

	return p.broker.conn.PublishMsg(natsMsg)
}

func (p *coreProducer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	for _, msg := range msgs {
		if err := p.Publish(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *coreProducer) Close() error {
	return nil
}

// jetStreamProducer is a JetStream producer with persistence.
type jetStreamProducer struct {
	broker  *Broker
	subject string
}

func (p *jetStreamProducer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	natsMsg := &nats.Msg{
		Subject: p.subject,
		Data:    msg.Payload,
		Header:  nats.Header{},
	}

	natsMsg.Header.Set("Nats-Msg-Id", msg.ID)
	for k, v := range msg.Headers {
		natsMsg.Header.Set(k, v)
	}

	ack, err := p.broker.js.PublishMsg(ctx, natsMsg)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	msg.Metadata.Offset = int64(ack.Sequence)

	return nil
}

func (p *jetStreamProducer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	for _, msg := range msgs {
		if err := p.Publish(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *jetStreamProducer) Close() error {
	return nil
}

// coreConsumer is a core NATS consumer (no persistence).
type coreConsumer struct {
	broker  *Broker
	subject string
	group   string
	sub     *nats.Subscription
	mu      sync.Mutex
}

func (c *coreConsumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	var sub *nats.Subscription
	var err error

	if c.group != "" {
		sub, err = c.broker.conn.QueueSubscribe(c.subject, c.group, func(natsMsg *nats.Msg) {
			msg := convertNatsMessage(natsMsg)
			handler(ctx, msg)
		})
	} else {
		sub, err = c.broker.conn.Subscribe(c.subject, func(natsMsg *nats.Msg) {
			msg := convertNatsMessage(natsMsg)
			handler(ctx, msg)
		})
	}

	if err != nil {
		return messaging.ErrConsumeFailed(err)
	}

	c.mu.Lock()
	c.sub = sub
	c.mu.Unlock()

	<-ctx.Done()
	return nil
}

func (c *coreConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sub != nil {
		return c.sub.Unsubscribe()
	}
	return nil
}

// jetStreamConsumer is a JetStream consumer with persistence.
type jetStreamConsumer struct {
	broker   *Broker
	consumer jetstream.Consumer
	subject  string
	group    string
	mu       sync.Mutex
	closed   bool
}

func (c *jetStreamConsumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	iter, err := c.consumer.Messages()
	if err != nil {
		return messaging.ErrConsumeFailed(err)
	}
	defer iter.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			natsMsg, err := iter.Next()
			if err != nil {
				if c.closed {
					return nil
				}
				continue
			}

			msg := convertJetStreamMessage(natsMsg)
			err = handler(ctx, msg)
			if err != nil {
				natsMsg.Nak()
			} else {
				natsMsg.Ack()
			}
		}
	}
}

func (c *jetStreamConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return nil
}

func convertNatsMessage(natsMsg *nats.Msg) *messaging.Message {
	msg := &messaging.Message{
		ID:        natsMsg.Header.Get("Nats-Msg-Id"),
		Topic:     natsMsg.Subject,
		Payload:   natsMsg.Data,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Metadata: messaging.MessageMetadata{
			Raw: natsMsg,
		},
	}

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	for k, v := range natsMsg.Header {
		if len(v) > 0 && k != "Nats-Msg-Id" {
			msg.Headers[k] = v[0]
		}
	}

	return msg
}

func convertJetStreamMessage(natsMsg jetstream.Msg) *messaging.Message {
	metadata, _ := natsMsg.Metadata()

	msg := &messaging.Message{
		ID:        natsMsg.Headers().Get("Nats-Msg-Id"),
		Topic:     natsMsg.Subject(),
		Payload:   natsMsg.Data(),
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Metadata: messaging.MessageMetadata{
			Offset: int64(metadata.Sequence.Stream),
			Raw:    natsMsg,
		},
	}

	if msg.ID == "" {
		msg.ID = fmt.Sprintf("%d", metadata.Sequence.Stream)
	}

	if metadata != nil {
		msg.Metadata.DeliveryCount = int(metadata.NumDelivered)
	}

	for k, v := range natsMsg.Headers() {
		if len(v) > 0 && k != "Nats-Msg-Id" {
			msg.Headers[k] = v[0]
		}
	}

	return msg
}
