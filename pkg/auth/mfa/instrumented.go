package mfa

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedProvider wraps Provider with observability.
type InstrumentedProvider struct {
	next   Provider
	tracer trace.Tracer
}

// NewInstrumentedProvider creates a new InstrumentedProvider.
func NewInstrumentedProvider(next Provider) *InstrumentedProvider {
	return &InstrumentedProvider{
		next:   next,
		tracer: otel.Tracer("pkg/auth/mfa"),
	}
}

// Enroll instruments Enroll.
func (p *InstrumentedProvider) Enroll(ctx context.Context, userID string) (string, []string, error) {
	ctx, span := p.tracer.Start(ctx, "mfa.Enroll", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "mfa enroll start", "user_id", userID)

	secret, recovery, err := p.next.Enroll(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa enroll failed", "error", err, "user_id", userID)
	} else {
		logger.L().InfoContext(ctx, "mfa enroll success", "user_id", userID)
	}

	return secret, recovery, err
}

// CompleteEnrollment instruments CompleteEnrollment.
func (p *InstrumentedProvider) CompleteEnrollment(ctx context.Context, userID, code string) error {
	ctx, span := p.tracer.Start(ctx, "mfa.CompleteEnrollment", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "mfa complete enrollment", "user_id", userID)

	err := p.next.CompleteEnrollment(ctx, userID, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa complete enrollment failed", "error", err, "user_id", userID)
	}

	return err
}

// Verify instruments Verify.
func (p *InstrumentedProvider) Verify(ctx context.Context, userID, code string) (bool, error) {
	ctx, span := p.tracer.Start(ctx, "mfa.Verify", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	valid, err := p.next.Verify(ctx, userID, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa verify failed", "error", err, "user_id", userID)
	} else if !valid {
		logger.L().WarnContext(ctx, "mfa verify invalid code", "user_id", userID)
	}

	return valid, err
}

// Recover instruments Recover.
func (p *InstrumentedProvider) Recover(ctx context.Context, userID, code string) (bool, error) {
	ctx, span := p.tracer.Start(ctx, "mfa.Recover", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "mfa recover attempt", "user_id", userID)

	success, err := p.next.Recover(ctx, userID, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa recover failed", "error", err, "user_id", userID)
	}

	return success, err
}

// Disable instruments Disable.
func (p *InstrumentedProvider) Disable(ctx context.Context, userID string) error {
	ctx, span := p.tracer.Start(ctx, "mfa.Disable", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "mfa disable", "user_id", userID)

	err := p.next.Disable(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa disable failed", "error", err, "user_id", userID)
	}

	return err
}
