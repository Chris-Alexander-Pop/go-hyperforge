package iot

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedClient implements Client at compile time.
var _ Client = (*EventedClient)(nil)

const (
	// TopicIoT is the pkg/events topic for IoT domain events.
	TopicIoT = "iot"

	// EventTypePublished is emitted after a successful Publish / PublishWithOptions.
	EventTypePublished = "iot.message.published"

	// EventTypeSubscribed is emitted after a successful Subscribe / SubscribeWithQoS.
	EventTypeSubscribed = "iot.subscribed"
)

// MessageEventPayload is the typed payload for IoT publish events.
type MessageEventPayload struct {
	Topic      string    `json:"topic"`
	PayloadLen int       `json:"payload_len"`
	Timestamp  time.Time `json:"timestamp"`
}

// SubscribeEventPayload is the typed payload for IoT subscribe events.
type SubscribeEventPayload struct {
	Topic     string    `json:"topic"`
	Timestamp time.Time `json:"timestamp"`
}

// EventedClient decorates a Client to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so MQTT ops are not rolled back.
type EventedClient struct {
	next Client
	bus  events.Bus
}

// NewEventedClient wraps next so Publish/Subscribe fan out to bus after success.
// If bus is nil, publishing is skipped.
func NewEventedClient(next Client, bus events.Bus) *EventedClient {
	return &EventedClient{next: next, bus: bus}
}

func (c *EventedClient) publishEvent(ctx context.Context, eventType string, payload interface{}) {
	if c.bus == nil {
		return
	}
	_ = c.bus.Publish(ctx, TopicIoT, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/iot",
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// Connect delegates to the underlying client.
func (c *EventedClient) Connect(ctx context.Context) error {
	return c.next.Connect(ctx)
}

// Disconnect delegates to the underlying client.
func (c *EventedClient) Disconnect() {
	c.next.Disconnect()
}

// IsConnected delegates to the underlying client.
func (c *EventedClient) IsConnected() bool {
	return c.next.IsConnected()
}

// Publish delegates then publishes iot.message.published (best-effort).
func (c *EventedClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if err := c.next.Publish(ctx, topic, payload); err != nil {
		return err
	}
	c.publishEvent(ctx, EventTypePublished, MessageEventPayload{
		Topic:      topic,
		PayloadLen: len(payload),
		Timestamp:  time.Now().UTC(),
	})
	return nil
}

// PublishWithOptions delegates then publishes iot.message.published (best-effort).
func (c *EventedClient) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos QoS, retained bool) error {
	if err := c.next.PublishWithOptions(ctx, topic, payload, qos, retained); err != nil {
		return err
	}
	c.publishEvent(ctx, EventTypePublished, MessageEventPayload{
		Topic:      topic,
		PayloadLen: len(payload),
		Timestamp:  time.Now().UTC(),
	})
	return nil
}

// Subscribe delegates then publishes iot.subscribed (best-effort).
func (c *EventedClient) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	if err := c.next.Subscribe(ctx, topic, handler); err != nil {
		return err
	}
	c.publishEvent(ctx, EventTypeSubscribed, SubscribeEventPayload{
		Topic:     topic,
		Timestamp: time.Now().UTC(),
	})
	return nil
}

// SubscribeWithQoS delegates then publishes iot.subscribed (best-effort).
func (c *EventedClient) SubscribeWithQoS(ctx context.Context, topic string, qos QoS, handler MessageHandler) error {
	if err := c.next.SubscribeWithQoS(ctx, topic, qos, handler); err != nil {
		return err
	}
	c.publishEvent(ctx, EventTypeSubscribed, SubscribeEventPayload{
		Topic:     topic,
		Timestamp: time.Now().UTC(),
	})
	return nil
}

// Unsubscribe delegates to the underlying client.
func (c *EventedClient) Unsubscribe(ctx context.Context, topic string) error {
	return c.next.Unsubscribe(ctx, topic)
}
