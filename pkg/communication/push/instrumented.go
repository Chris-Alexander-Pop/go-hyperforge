package push

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// InstrumentedSender is a wrapper around a Sender that adds observability.
type InstrumentedSender struct {
	next   Sender
	tracer trace.Tracer
}

// NewInstrumentedSender creates a new InstrumentedSender.
func NewInstrumentedSender(next Sender) *InstrumentedSender {
	return &InstrumentedSender{
		next:   next,
		tracer: otel.Tracer("pkg/communication/push"),
	}
}

// Send dispatches a single push notification with observability.
func (s *InstrumentedSender) Send(ctx context.Context, msg *Message) error {
	ctx, span := s.tracer.Start(ctx, "push.Send", trace.WithAttributes(
		attribute.String("push.title", msg.Title),
		attribute.Int("push.tokens_count", len(msg.Tokens)),
		attribute.String("push.platform", msg.Platform),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending push notification",
		"title", msg.Title,
		"tokens_count", len(msg.Tokens),
	)

	err := s.next.Send(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send push notification",
			"error", err,
			"title", msg.Title,
			"tokens_count", len(msg.Tokens),
		)
	}

	return err
}

// SendBatch dispatches multiple push notifications with observability.
func (s *InstrumentedSender) SendBatch(ctx context.Context, msgs []*Message) error {
	ctx, span := s.tracer.Start(ctx, "push.SendBatch", trace.WithAttributes(
		attribute.Int("push.messages_count", len(msgs)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending push notification batch",
		"messages_count", len(msgs),
	)

	err := s.next.SendBatch(ctx, msgs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send push notification batch",
			"error", err,
			"messages_count", len(msgs),
		)
	}

	return err
}

// Close releases any resources held by the sender.
func (s *InstrumentedSender) Close() error {
	return s.next.Close()
}
