package validator

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure InstrumentedValidator implements Validator at compile time.
var _ Validator = (*InstrumentedValidator)(nil)

// InstrumentedValidator wraps a Validator with logging and OpenTelemetry tracing.
type InstrumentedValidator struct {
	next   Validator
	tracer trace.Tracer
}

// NewInstrumentedValidator decorates next with logging and tracing.
func NewInstrumentedValidator(next Validator) *InstrumentedValidator {
	return &InstrumentedValidator{
		next:   next,
		tracer: otel.Tracer("pkg/validator"),
	}
}

// ValidateStruct validates a struct under a trace span.
func (v *InstrumentedValidator) ValidateStruct(ctx context.Context, s interface{}) error {
	ctx, span := v.tracer.Start(ctx, "validator.ValidateStruct")
	defer span.End()

	err := v.next.ValidateStruct(ctx, s)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "struct validation failed", "error", err)
		return err
	}
	return nil
}

// ValidateVar validates a single variable under a trace span.
func (v *InstrumentedValidator) ValidateVar(ctx context.Context, field interface{}, tag string) error {
	ctx, span := v.tracer.Start(ctx, "validator.ValidateVar",
		trace.WithAttributes(attribute.String("validator.tag", tag)),
	)
	defer span.End()

	err := v.next.ValidateVar(ctx, field, tag)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "variable validation failed", "tag", tag, "error", err)
		return err
	}
	return nil
}
