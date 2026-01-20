package email

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
		tracer: otel.Tracer("pkg/communication/email"),
	}
}

// Send dispatches a single email message with observability.
func (s *InstrumentedSender) Send(ctx context.Context, msg *Message) error {
	ctx, span := s.tracer.Start(ctx, "email.Send", trace.WithAttributes(
		attribute.String("email.subject", msg.Subject),
		attribute.Int("email.recipients_count", len(msg.To)+len(msg.CC)+len(msg.BCC)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending email",
		"subject", msg.Subject,
		"recipients_count", len(msg.To)+len(msg.CC)+len(msg.BCC),
	)

	err := s.next.Send(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send email",
			"error", err,
			"subject", msg.Subject,
		)
	}

	return err
}

// SendBatch dispatches multiple email messages with observability.
func (s *InstrumentedSender) SendBatch(ctx context.Context, msgs []*Message) error {
	ctx, span := s.tracer.Start(ctx, "email.SendBatch", trace.WithAttributes(
		attribute.Int("email.messages_count", len(msgs)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "sending email batch",
		"messages_count", len(msgs),
	)

	err := s.next.SendBatch(ctx, msgs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to send email batch",
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
