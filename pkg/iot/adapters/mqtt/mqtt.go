// Package mqtt wraps protocols/mqtt (Eclipse Paho) behind pkg/iot.Client.
//
// Prefer this adapter (or adapters/memory) for interface-based consumers.
// protocols/mqtt remains the concrete Paho client; this package adapts types.
package mqtt

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	protomqtt "github.com/chris-alexander-pop/go-hyperforge/pkg/iot/protocols/mqtt"
)

// Ensure Adapter implements iot.Client.
var _ iot.Client = (*Adapter)(nil)

// Config is an alias for the underlying Paho client config.
type Config = protomqtt.Config

// TLSConfig is an alias for MQTT TLS settings.
type TLSConfig = protomqtt.TLSConfig

// Adapter implements iot.Client over protocols/mqtt.Client.
type Adapter struct {
	inner *protomqtt.Client
}

// New creates a Paho-backed iot.Client.
func New(cfg Config) (*Adapter, error) {
	inner, err := protomqtt.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Adapter{inner: inner}, nil
}

// NewFromClient wraps an existing protocols/mqtt.Client.
func NewFromClient(c *protomqtt.Client) (*Adapter, error) {
	if c == nil {
		return nil, iot.ErrInvalidConfig("mqtt client is required", nil)
	}
	return &Adapter{inner: c}, nil
}

// Connect establishes a connection to the broker.
func (a *Adapter) Connect(ctx context.Context) error {
	return a.inner.Connect(ctx)
}

// Disconnect closes the connection.
func (a *Adapter) Disconnect() {
	a.inner.Disconnect()
}

// IsConnected reports whether the client is currently connected.
func (a *Adapter) IsConnected() bool {
	return a.inner.IsConnected()
}

// Publish sends a message with QoS at-least-once and retain=false.
func (a *Adapter) Publish(ctx context.Context, topic string, payload []byte) error {
	return a.inner.Publish(ctx, topic, payload)
}

// PublishWithOptions sends a message with explicit QoS and retain settings.
func (a *Adapter) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos iot.QoS, retained bool) error {
	return a.inner.PublishWithOptions(ctx, topic, payload, protomqtt.QoS(qos), retained)
}

// Subscribe registers a handler for a topic (QoS at-least-once).
func (a *Adapter) Subscribe(ctx context.Context, topic string, handler iot.MessageHandler) error {
	return a.SubscribeWithQoS(ctx, topic, iot.QoSAtLeastOnce, handler)
}

// SubscribeWithQoS subscribes with a specific QoS level.
func (a *Adapter) SubscribeWithQoS(ctx context.Context, topic string, qos iot.QoS, handler iot.MessageHandler) error {
	return a.inner.SubscribeWithQoS(ctx, topic, protomqtt.QoS(qos), adaptHandler(handler))
}

// Unsubscribe removes a topic subscription.
func (a *Adapter) Unsubscribe(ctx context.Context, topic string) error {
	return a.inner.Unsubscribe(ctx, topic)
}

func adaptHandler(h iot.MessageHandler) protomqtt.MessageHandler {
	if h == nil {
		return nil
	}
	return func(msg *protomqtt.Message) {
		if msg == nil {
			return
		}
		h(&iot.Message{
			Topic:     msg.Topic,
			Payload:   msg.Payload,
			QoS:       iot.QoS(msg.QoS),
			Retained:  msg.Retained,
			MessageID: msg.MessageID,
		})
	}
}
