package payment

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedProvider wraps a Provider with logging and tracing.
type InstrumentedProvider struct {
	next   Provider
	tracer trace.Tracer
}

// NewInstrumentedProvider creates a new InstrumentedProvider.
func NewInstrumentedProvider(next Provider) *InstrumentedProvider {
	return &InstrumentedProvider{
		next:   next,
		tracer: otel.Tracer("pkg/commerce/payment"),
	}
}

func (p *InstrumentedProvider) Charge(ctx context.Context, req *ChargeRequest) (*Transaction, error) {
	ctx, span := p.tracer.Start(ctx, "payment.Charge", trace.WithAttributes(
		attribute.Float64("charge.amount", req.Amount),
		attribute.String("charge.currency", req.Currency),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "processing charge", "amount", req.Amount, "currency", req.Currency)

	tx, err := p.next.Charge(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "charge failed", "error", err)
	} else {
		span.SetAttributes(attribute.String("transaction.id", tx.ID))
		logger.L().InfoContext(ctx, "charge succeeded", "transaction_id", tx.ID)
	}
	return tx, err
}

func (p *InstrumentedProvider) Refund(ctx context.Context, req *RefundRequest) (*Transaction, error) {
	ctx, span := p.tracer.Start(ctx, "payment.Refund", trace.WithAttributes(
		attribute.String("transaction.id", req.TransactionID),
		attribute.Float64("refund.amount", req.Amount),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "processing refund", "transaction_id", req.TransactionID)

	tx, err := p.next.Refund(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "refund failed", "error", err)
	} else {
		logger.L().InfoContext(ctx, "refund succeeded", "transaction_id", tx.ID)
	}
	return tx, err
}

func (p *InstrumentedProvider) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	ctx, span := p.tracer.Start(ctx, "payment.GetTransaction", trace.WithAttributes(
		attribute.String("transaction.id", id),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "retrieving transaction", "id", id)

	tx, err := p.next.GetTransaction(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get transaction", "id", id, "error", err)
	}
	return tx, err
}

func (p *InstrumentedProvider) Close() error {
	return p.next.Close()
}
