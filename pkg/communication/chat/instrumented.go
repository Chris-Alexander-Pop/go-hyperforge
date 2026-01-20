package chat

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
		tracer: otel.Tracer("pkg/communication/chat"),
	}
}

// Send dispatches a single chat message with observability.
func (s *InstrumentedSender) Send(ctx context.Context, msg *Message) error {
	ctx, span := s.tracer.Start(ctx, "chat.Send", trace.WithAttributes(
		attribute.String("chat.channel_id", msg.ChannelID),
		attribute.String("chat.user_id", msg.UserID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending chat message",
		"channel_id", msg.ChannelID,
		"user_id", msg.UserID,
	)

	err := s.next.Send(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send chat message",
			"error", err,
			"channel_id", msg.ChannelID,
		)
	}

	return err
}

// Close releases any resources held by the sender.
func (s *InstrumentedSender) Close() error {
	return s.next.Close()
}
