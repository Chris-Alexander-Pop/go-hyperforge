package memory

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
)

// Ensure compile-time interface compliance.
var _ iot.Client = (*Client)(nil)

// Client is an in-memory MQTT client for tests and local development.
// Messages published to a topic are delivered synchronously to matching subscribers.
type Client struct {
	mu        *concurrency.SmartRWMutex
	handlers  map[string]iot.MessageHandler
	connected atomic.Bool
	msgID     atomic.Uint32
	closed    atomic.Bool
}

// NewClient creates a connected-ready in-memory MQTT client.
// Call Connect before Publish/Subscribe; Disconnect marks the client offline.
func NewClient() *Client {
	return &Client{
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "iot-memory-mqtt"}),
		handlers: make(map[string]iot.MessageHandler),
	}
}

// Connect marks the client as connected.
func (c *Client) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.closed.Load() {
		return iot.ErrNotConnected()
	}
	c.connected.Store(true)
	return nil
}

// Disconnect marks the client as disconnected.
func (c *Client) Disconnect() {
	c.connected.Store(false)
}

// IsConnected reports connection state.
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// Publish sends a message with QoS at-least-once.
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	return c.PublishWithOptions(ctx, topic, payload, iot.QoSAtLeastOnce, false)
}

// PublishWithOptions delivers the message to all matching local subscribers.
func (c *Client) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos iot.QoS, retained bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !c.connected.Load() {
		return iot.ErrNotConnected()
	}
	if topic == "" {
		return iot.ErrInvalidConfig("topic is required", nil)
	}

	id := uint16(c.msgID.Add(1))
	msg := &iot.Message{
		Topic:     topic,
		Payload:   append([]byte(nil), payload...),
		QoS:       qos,
		Retained:  retained,
		MessageID: id,
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for filter, handler := range c.handlers {
		if handler == nil {
			continue
		}
		if topicMatch(filter, topic) {
			handler(msg)
		}
	}
	return nil
}

// Subscribe registers a handler for a topic filter.
func (c *Client) Subscribe(ctx context.Context, topic string, handler iot.MessageHandler) error {
	return c.SubscribeWithQoS(ctx, topic, iot.QoSAtLeastOnce, handler)
}

// SubscribeWithQoS registers a handler for a topic filter at the given QoS.
func (c *Client) SubscribeWithQoS(ctx context.Context, topic string, qos iot.QoS, handler iot.MessageHandler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !c.connected.Load() {
		return iot.ErrNotConnected()
	}
	if topic == "" {
		return iot.ErrInvalidConfig("topic is required", nil)
	}
	if handler == nil {
		return iot.ErrInvalidConfig("handler is required", nil)
	}
	_ = qos // QoS is recorded on published messages; memory delivery is local.

	c.mu.Lock()
	c.handlers[topic] = handler
	c.mu.Unlock()
	return nil
}

// Unsubscribe removes a topic subscription.
func (c *Client) Unsubscribe(ctx context.Context, topic string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.handlers, topic)
	c.mu.Unlock()
	return nil
}

// topicMatch supports MQTT-style + and # wildcards (simplified).
func topicMatch(filter, topic string) bool {
	if filter == topic || filter == "#" {
		return true
	}
	fParts := strings.Split(filter, "/")
	tParts := strings.Split(topic, "/")
	for i := 0; i < len(fParts); i++ {
		if fParts[i] == "#" {
			return true
		}
		if i >= len(tParts) {
			return false
		}
		if fParts[i] == "+" {
			continue
		}
		if fParts[i] != tParts[i] {
			return false
		}
	}
	return len(fParts) == len(tParts)
}
