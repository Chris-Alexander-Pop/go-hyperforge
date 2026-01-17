package messaging

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedBroker wraps a Broker with logging and tracing.
type InstrumentedBroker struct {
	next   Broker
	tracer trace.Tracer
}

// NewInstrumentedBroker creates a new InstrumentedBroker wrapping the given broker.
func NewInstrumentedBroker(next Broker) *InstrumentedBroker {
	return &InstrumentedBroker{
		next:   next,
		tracer: otel.Tracer("pkg/messaging"),
	}
}

func (b *InstrumentedBroker) Producer(topic string) (Producer, error) {
	producer, err := b.next.Producer(topic)
	if err != nil {
		logger.L().Error("failed to create producer", "topic", topic, "error", err)
		return nil, err
	}
	return &InstrumentedProducer{
		next:   producer,
		topic:  topic,
		tracer: b.tracer,
	}, nil
}

func (b *InstrumentedBroker) Consumer(topic string, group string) (Consumer, error) {
	consumer, err := b.next.Consumer(topic, group)
	if err != nil {
		logger.L().Error("failed to create consumer", "topic", topic, "group", group, "error", err)
		return nil, err
	}
	return &InstrumentedConsumer{
		next:   consumer,
		topic:  topic,
		group:  group,
		tracer: b.tracer,
	}, nil
}

func (b *InstrumentedBroker) Close() error {
	logger.L().Info("closing messaging broker")
	return b.next.Close()
}

func (b *InstrumentedBroker) Healthy(ctx context.Context) bool {
	return b.next.Healthy(ctx)
}

// InstrumentedProducer wraps a Producer with logging and tracing.
type InstrumentedProducer struct {
	next   Producer
	topic  string
	tracer trace.Tracer
}

func (p *InstrumentedProducer) Publish(ctx context.Context, msg *Message) error {
	ctx, span := p.tracer.Start(ctx, "messaging.Publish", trace.WithAttributes(
		attribute.String("messaging.topic", p.topic),
		attribute.String("messaging.message_id", msg.ID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "publishing message", "topic", p.topic, "message_id", msg.ID)

	err := p.next.Publish(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to publish message", "topic", p.topic, "error", err)
		return err
	}

	span.SetStatus(codes.Ok, "message published")
	return nil
}

func (p *InstrumentedProducer) PublishBatch(ctx context.Context, msgs []*Message) error {
	ctx, span := p.tracer.Start(ctx, "messaging.PublishBatch", trace.WithAttributes(
		attribute.String("messaging.topic", p.topic),
		attribute.Int("messaging.batch_size", len(msgs)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "publishing message batch", "topic", p.topic, "batch_size", len(msgs))

	err := p.next.PublishBatch(ctx, msgs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to publish batch", "topic", p.topic, "error", err)
		return err
	}

	span.SetStatus(codes.Ok, "batch published")
	return nil
}

func (p *InstrumentedProducer) Close() error {
	logger.L().Info("closing producer", "topic", p.topic)
	return p.next.Close()
}

// InstrumentedConsumer wraps a Consumer with logging and tracing.
type InstrumentedConsumer struct {
	next   Consumer
	topic  string
	group  string
	tracer trace.Tracer
}

func (c *InstrumentedConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	logger.L().InfoContext(ctx, "starting consumer", "topic", c.topic, "group", c.group)

	// Wrap the handler to trace each message processing
	instrumentedHandler := func(ctx context.Context, msg *Message) error {
		ctx, span := c.tracer.Start(ctx, "messaging.HandleMessage", trace.WithAttributes(
			attribute.String("messaging.topic", c.topic),
			attribute.String("messaging.group", c.group),
			attribute.String("messaging.message_id", msg.ID),
		))
		defer span.End()

		logger.L().InfoContext(ctx, "processing message", "topic", c.topic, "message_id", msg.ID)

		err := handler(ctx, msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.L().ErrorContext(ctx, "failed to process message", "topic", c.topic, "message_id", msg.ID, "error", err)
			return err
		}

		span.SetStatus(codes.Ok, "message processed")
		return nil
	}

	return c.next.Consume(ctx, instrumentedHandler)
}

func (c *InstrumentedConsumer) Close() error {
	logger.L().Info("closing consumer", "topic", c.topic, "group", c.group)
	return c.next.Close()
}
