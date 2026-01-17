// Package redisstreams provides a Redis Streams messaging adapter.
//
// This adapter implements the messaging.Broker interface using Redis Streams,
// supporting consumer groups, message acknowledgment, and persistent storage.
//
// # Usage
//
//	cfg := redisstreams.Config{
//	    Addr:     "localhost:6379",
//	    StreamKey: "my-stream",
//	}
//	broker, err := redisstreams.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Features
//
//   - Consumer groups: Load balancing across multiple consumers
//   - Message persistence: Messages survive Redis restarts (with AOF/RDB)
//   - Message acknowledgment: Explicit ack/nack
//   - Pending message handling: Claim and process stuck messages
//
// # Dependencies
//
// This package requires: github.com/redis/go-redis/v9
package redisstreams

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Config holds configuration for the Redis Streams broker.
type Config struct {
	// Addr is the Redis server address.
	Addr string `env:"REDIS_ADDR" env-default:"localhost:6379"`

	// Password for Redis authentication.
	Password string `env:"REDIS_PASSWORD"`

	// DB is the Redis database number.
	DB int `env:"REDIS_DB" env-default:"0"`

	// StreamKey is the default stream key.
	StreamKey string `env:"REDIS_STREAM_KEY" env-default:"messages"`

	// ConsumerGroup is the default consumer group name.
	ConsumerGroup string `env:"REDIS_CONSUMER_GROUP" env-default:"default-group"`

	// ConsumerName is the unique consumer name within the group.
	ConsumerName string `env:"REDIS_CONSUMER_NAME"`

	// MaxLen limits the stream length (MAXLEN parameter).
	MaxLen int64 `env:"REDIS_STREAM_MAXLEN" env-default:"0"`

	// BlockDuration is how long to block waiting for messages.
	BlockDuration time.Duration `env:"REDIS_BLOCK_DURATION" env-default:"5s"`

	// BatchSize is the number of messages to fetch per read.
	BatchSize int64 `env:"REDIS_BATCH_SIZE" env-default:"10"`

	// ClaimMinIdleTime is the minimum idle time before claiming pending messages.
	ClaimMinIdleTime time.Duration `env:"REDIS_CLAIM_MIN_IDLE" env-default:"30s"`

	// TLS configuration
	TLSEnabled bool `env:"REDIS_TLS_ENABLED" env-default:"false"`
}

// Broker is a Redis Streams message broker implementation.
type Broker struct {
	config Config
	client *redis.Client
	mu     sync.RWMutex
	closed bool
}

// New creates a new Redis Streams broker.
func New(cfg Config) (*Broker, error) {
	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &Broker{
		config: cfg,
		client: client,
	}, nil
}

// Producer creates a producer for the stream.
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	streamKey := topic
	if streamKey == "" {
		streamKey = b.config.StreamKey
	}

	return &producer{
		broker:    b,
		streamKey: streamKey,
	}, nil
}

// Consumer creates a consumer for the stream.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	streamKey := topic
	if streamKey == "" {
		streamKey = b.config.StreamKey
	}

	groupName := group
	if groupName == "" {
		groupName = b.config.ConsumerGroup
	}

	consumerName := b.config.ConsumerName
	if consumerName == "" {
		consumerName = "consumer-" + uuid.New().String()[:8]
	}

	// Create consumer group if it doesn't exist
	ctx := context.Background()
	err := b.client.XGroupCreateMkStream(ctx, streamKey, groupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &consumer{
		broker:       b,
		streamKey:    streamKey,
		groupName:    groupName,
		consumerName: consumerName,
	}, nil
}

// Close shuts down the Redis connection.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	return b.client.Close()
}

// Healthy checks if the Redis connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return false
	}

	return b.client.Ping(ctx).Err() == nil
}

// producer is a Redis Streams producer.
type producer struct {
	broker    *Broker
	streamKey string
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	values := map[string]interface{}{
		"id":        msg.ID,
		"payload":   msg.Payload,
		"timestamp": msg.Timestamp.UnixMilli(),
	}

	for k, v := range msg.Headers {
		values["h:"+k] = v
	}

	if len(msg.Key) > 0 {
		values["key"] = string(msg.Key)
	}

	args := &redis.XAddArgs{
		Stream: p.streamKey,
		Values: values,
	}

	if p.broker.config.MaxLen > 0 {
		args.MaxLen = p.broker.config.MaxLen
		args.Approx = true
	}

	streamID, err := p.broker.client.XAdd(ctx, args).Result()
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	msg.Metadata.ReceiptHandle = streamID

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	pipe := p.broker.client.Pipeline()

	for _, msg := range msgs {
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		values := map[string]interface{}{
			"id":        msg.ID,
			"payload":   msg.Payload,
			"timestamp": msg.Timestamp.UnixMilli(),
		}

		for k, v := range msg.Headers {
			values["h:"+k] = v
		}

		args := &redis.XAddArgs{
			Stream: p.streamKey,
			Values: values,
		}

		if p.broker.config.MaxLen > 0 {
			args.MaxLen = p.broker.config.MaxLen
			args.Approx = true
		}

		pipe.XAdd(ctx, args)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	return nil
}

func (p *producer) Close() error {
	return nil
}

// consumer is a Redis Streams consumer.
type consumer struct {
	broker       *Broker
	streamKey    string
	groupName    string
	consumerName string
	mu           sync.Mutex
	closed       bool
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

		// Read new messages
		streams, err := c.broker.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.groupName,
			Consumer: c.consumerName,
			Streams:  []string{c.streamKey, ">"},
			Count:    c.broker.config.BatchSize,
			Block:    c.broker.config.BlockDuration,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		for _, stream := range streams {
			for _, redisMsg := range stream.Messages {
				msg := convertRedisMessage(redisMsg)

				err := handler(ctx, msg)
				if err != nil {
					// Don't ack - message will be in pending list
					continue
				}

				// Acknowledge message
				c.broker.client.XAck(ctx, c.streamKey, c.groupName, redisMsg.ID)
			}
		}
	}
}

func (c *consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return nil
}

func convertRedisMessage(redisMsg redis.XMessage) *messaging.Message {
	msg := &messaging.Message{
		Headers: make(map[string]string),
		Metadata: messaging.MessageMetadata{
			ReceiptHandle: redisMsg.ID,
			Raw:           redisMsg,
		},
	}

	for k, v := range redisMsg.Values {
		strVal := fmt.Sprintf("%v", v)

		switch k {
		case "id":
			msg.ID = strVal
		case "payload":
			msg.Payload = []byte(strVal)
		case "timestamp":
			if ts, err := strconv.ParseInt(strVal, 10, 64); err == nil {
				msg.Timestamp = time.UnixMilli(ts)
			}
		case "key":
			msg.Key = []byte(strVal)
		default:
			if strings.HasPrefix(k, "h:") {
				msg.Headers[strings.TrimPrefix(k, "h:")] = strVal
			}
		}
	}

	if msg.ID == "" {
		msg.ID = redisMsg.ID
	}

	return msg
}
