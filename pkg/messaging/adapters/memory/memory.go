// Package memory provides an in-memory messaging implementation for testing.
//
// This adapter uses Go channels to simulate a message broker, making it ideal
// for unit tests and local development without external dependencies.
//
// # Usage
//
//	broker := memory.New(memory.Config{BufferSize: 100})
//	defer broker.Close()
//
//	producer, _ := broker.Producer("my-topic")
//	consumer, _ := broker.Consumer("my-topic", "my-group")
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
)

// Config holds configuration for the memory broker.
type Config struct {
	// BufferSize is the channel buffer size for each topic.
	// Larger values allow more messages to be buffered before blocking.
	BufferSize int `env:"MEMORY_BUFFER_SIZE" env-default:"1000"`
}

// Broker is an in-memory message broker implementation.
type Broker struct {
	config Config
	mu     sync.RWMutex
	topics map[string]*topic
	closed bool
}

type topic struct {
	mu          sync.RWMutex
	name        string
	subscribers map[string]chan *messaging.Message // group -> channel
}

// New creates a new in-memory broker.
func New(cfg Config) *Broker {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	return &Broker{
		config: cfg,
		topics: make(map[string]*topic),
	}
}

func (b *Broker) getOrCreateTopic(name string) *topic {
	b.mu.Lock()
	defer b.mu.Unlock()

	if t, ok := b.topics[name]; ok {
		return t
	}

	t := &topic{
		name:        name,
		subscribers: make(map[string]chan *messaging.Message),
	}
	b.topics[name] = t
	return t
}

// Producer creates a new producer for the specified topic.
func (b *Broker) Producer(topicName string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	t := b.getOrCreateTopic(topicName)
	return &producer{
		broker: b,
		topic:  t,
	}, nil
}

// Consumer creates a new consumer for the specified topic and group.
func (b *Broker) Consumer(topicName string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	if group == "" {
		group = uuid.New().String() // Unique group for broadcast
	}

	t := b.getOrCreateTopic(topicName)

	t.mu.Lock()
	ch := make(chan *messaging.Message, b.config.BufferSize)
	t.subscribers[group] = ch
	t.mu.Unlock()

	return &consumer{
		broker: b,
		topic:  t,
		group:  group,
		ch:     ch,
	}, nil
}

// Close shuts down the broker and all topics.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	for _, t := range b.topics {
		t.mu.Lock()
		for _, ch := range t.subscribers {
			close(ch)
		}
		t.mu.Unlock()
	}

	return nil
}

// Healthy returns true if the broker is operational.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return !b.closed
}

// producer is an in-memory message producer.
type producer struct {
	broker *Broker
	topic  *topic
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	if msg.Topic == "" {
		msg.Topic = p.topic.name
	}

	p.topic.mu.RLock()
	defer p.topic.mu.RUnlock()

	// Fan-out to all subscribers
	for _, ch := range p.topic.subscribers {
		select {
		case ch <- msg:
		case <-ctx.Done():
			return messaging.ErrTimeout("publish", ctx.Err())
		default:
			// Channel full, skip (or could return error)
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
	return nil
}

// consumer is an in-memory message consumer.
type consumer struct {
	broker *Broker
	topic  *topic
	group  string
	ch     chan *messaging.Message
	closed bool
	mu     sync.Mutex
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-c.ch:
			if !ok {
				return nil // Channel closed
			}
			if err := handler(ctx, msg); err != nil {
				// In memory, we just log and continue
				// Real implementations might requeue
				continue
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

	c.topic.mu.Lock()
	delete(c.topic.subscribers, c.group)
	c.topic.mu.Unlock()

	return nil
}
