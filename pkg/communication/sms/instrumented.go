package sms

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
		tracer: otel.Tracer("pkg/communication/sms"),
	}
}

// Send dispatches a single SMS message with observability.
func (s *InstrumentedSender) Send(ctx context.Context, msg *Message) error {
	ctx, span := s.tracer.Start(ctx, "sms.Send", trace.WithAttributes(
		attribute.String("sms.to", msg.To),
		attribute.String("sms.from", msg.From),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending sms",
		"to", msg.To,
		"from", msg.From,
	)

	err := s.next.Send(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send sms",
			"error", err,
			"to", msg.To,
		)
	}

	return err
}

// SendBatch dispatches multiple SMS messages with observability.
func (s *InstrumentedSender) SendBatch(ctx context.Context, msgs []*Message) error {
	ctx, span := s.tracer.Start(ctx, "sms.SendBatch", trace.WithAttributes(
		attribute.Int("sms.messages_count", len(msgs)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending sms batch",
		"messages_count", len(msgs),
	)

	err := s.next.SendBatch(ctx, msgs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send sms batch",
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
