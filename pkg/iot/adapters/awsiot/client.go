package awsiot

import (
	"context"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
)

// Ensure Adapter implements iot.Client.
var _ iot.Client = (*Adapter)(nil)

// Publisher abstracts AWS IoT data-plane publish for testability.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

// Adapter wraps an AWS IoT publisher behind pkg/iot.Client.
//
// Publish methods forward to the AWS data plane. Subscribe is handled in-process
// so locally published messages fan out to matching handlers (useful for tests
// and single-process bridges). Remote MQTT subscribe requires protocols/mqtt
// with AWS IoT credentials.
type Adapter struct {
	pub       Publisher
	mu        *concurrency.SmartRWMutex
	handlers  map[string]iot.MessageHandler
	connected atomic.Bool
	msgID     atomic.Uint32
}

// NewAdapter creates an iot.Client adapter over pub.
func NewAdapter(pub Publisher) (*Adapter, error) {
	if pub == nil {
		return nil, iot.ErrInvalidConfig("publisher is required", nil)
	}
	return &Adapter{
		pub:      pub,
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "iot-awsiot-adapter"}),
		handlers: make(map[string]iot.MessageHandler),
	}, nil
}

// NewAdapterFromClient wraps a concrete *Client as iot.Client.
func NewAdapterFromClient(c *Client) (*Adapter, error) {
	if c == nil {
		return nil, iot.ErrInvalidConfig("aws iot client is required", nil)
	}
	return NewAdapter(c)
}

// Connect marks the adapter connected.
func (a *Adapter) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.connected.Store(true)
	return nil
}

// Disconnect marks the adapter disconnected.
func (a *Adapter) Disconnect() {
	a.connected.Store(false)
}

// IsConnected reports connection state.
func (a *Adapter) IsConnected() bool {
	return a.connected.Load()
}

// Publish sends a message via AWS IoT and fans out to local subscribers.
func (a *Adapter) Publish(ctx context.Context, topic string, payload []byte) error {
	return a.PublishWithOptions(ctx, topic, payload, iot.QoSAtLeastOnce, false)
}

// PublishWithOptions publishes to AWS IoT then notifies local handlers.
func (a *Adapter) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos iot.QoS, retained bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.connected.Load() {
		return iot.ErrNotConnected()
	}
	if topic == "" {
		return iot.ErrInvalidConfig("topic is required", nil)
	}
	if err := a.pub.Publish(ctx, topic, payload); err != nil {
		return iot.ErrPublishFailed(err)
	}

	id := uint16(a.msgID.Add(1))
	msg := &iot.Message{
		Topic:     topic,
		Payload:   append([]byte(nil), payload...),
		QoS:       qos,
		Retained:  retained,
		MessageID: id,
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	for filter, handler := range a.handlers {
		if handler == nil {
			continue
		}
		if topicMatch(filter, topic) {
			handler(msg)
		}
	}
	return nil
}

// Subscribe registers a local topic handler.
func (a *Adapter) Subscribe(ctx context.Context, topic string, handler iot.MessageHandler) error {
	return a.SubscribeWithQoS(ctx, topic, iot.QoSAtLeastOnce, handler)
}

// SubscribeWithQoS registers a local topic handler (QoS recorded on delivered messages).
func (a *Adapter) SubscribeWithQoS(ctx context.Context, topic string, qos iot.QoS, handler iot.MessageHandler) error {
	_ = qos
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.connected.Load() {
		return iot.ErrNotConnected()
	}
	if topic == "" || handler == nil {
		return iot.ErrInvalidConfig("topic and handler are required", nil)
	}
	a.mu.Lock()
	a.handlers[topic] = handler
	a.mu.Unlock()
	return nil
}

// Unsubscribe removes a local topic handler.
func (a *Adapter) Unsubscribe(ctx context.Context, topic string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	delete(a.handlers, topic)
	a.mu.Unlock()
	return nil
}

func topicMatch(filter, topic string) bool {
	if filter == topic || filter == "#" {
		return true
	}
	// Minimal MQTT single-level / multi-level support for common filters.
	fp := splitTopic(filter)
	tp := splitTopic(topic)
	for i := 0; i < len(fp); i++ {
		if fp[i] == "#" {
			return true
		}
		if i >= len(tp) {
			return false
		}
		if fp[i] == "+" {
			continue
		}
		if fp[i] != tp[i] {
			return false
		}
	}
	return len(fp) == len(tp)
}

func splitTopic(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0, 4)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
