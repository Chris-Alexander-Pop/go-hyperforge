package mfa

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedProvider wraps a Provider with observability.
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

func (p *InstrumentedProvider) Enroll(ctx context.Context, userID string) (string, []string, error) {
	ctx, span := p.tracer.Start(ctx, "mfa.Enroll", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "initiating mfa enrollment", "user_id", userID)

	secret, recovery, err := p.next.Enroll(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa enrollment failed", "error", err, "user_id", userID)
		return "", nil, err
	}
	return secret, recovery, nil
}

func (p *InstrumentedProvider) CompleteEnrollment(ctx context.Context, userID, code string) error {
	ctx, span := p.tracer.Start(ctx, "mfa.CompleteEnrollment", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "completing mfa enrollment", "user_id", userID)

	err := p.next.CompleteEnrollment(ctx, userID, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "mfa completion failed", "error", err, "user_id", userID)
		return err
	}
	logger.L().InfoContext(ctx, "mfa enrollment completed", "user_id", userID)
	return nil
}

func (p *InstrumentedProvider) Verify(ctx context.Context, userID, code string) (bool, error) {
	ctx, span := p.tracer.Start(ctx, "mfa.Verify", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "verifying mfa code", "user_id", userID)

	valid, err := p.next.Verify(ctx, userID, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa verification error", "error", err, "user_id", userID)
		return false, err
	}
	span.SetAttributes(attribute.Bool("mfa.valid", valid))
	if !valid {
		logger.L().WarnContext(ctx, "invalid mfa code", "user_id", userID)
	}
	return valid, nil
}

func (p *InstrumentedProvider) Recover(ctx context.Context, userID, code string) (bool, error) {
	ctx, span := p.tracer.Start(ctx, "mfa.Recover", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "attempting mfa recovery", "user_id", userID)

	valid, err := p.next.Recover(ctx, userID, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "mfa recovery error", "error", err, "user_id", userID)
		return false, err
	}
	if valid {
		logger.L().WarnContext(ctx, "mfa recovery successful", "user_id", userID)
	} else {
		logger.L().WarnContext(ctx, "invalid recovery code", "user_id", userID)
	}
	return valid, nil
}

func (p *InstrumentedProvider) Disable(ctx context.Context, userID string) error {
	ctx, span := p.tracer.Start(ctx, "mfa.Disable", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "disabling mfa", "user_id", userID)

	err := p.next.Disable(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to disable mfa", "error", err, "user_id", userID)
		return err
	}
	return nil
}
