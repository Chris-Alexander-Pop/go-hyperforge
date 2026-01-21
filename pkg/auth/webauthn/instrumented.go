package webauthn

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedService wraps a Service with observability.
type InstrumentedService struct {
	next   Service
	tracer trace.Tracer
}

// NewInstrumentedService creates a new InstrumentedService.
func NewInstrumentedService(next Service) *InstrumentedService {
	return &InstrumentedService{
		next:   next,
		tracer: otel.Tracer("pkg/auth/webauthn"),
	}
}

func (s *InstrumentedService) BeginRegistration(ctx context.Context, user User) (interface{}, error) {
	ctx, span := s.tracer.Start(ctx, "webauthn.BeginRegistration", trace.WithAttributes(
		attribute.String("user.name", user.WebAuthnName()),
	))
	defer span.End()

	res, err := s.next.BeginRegistration(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "webauthn begin registration failed", "error", err)
		return nil, err
	}
	return res, nil
}

func (s *InstrumentedService) FinishRegistration(ctx context.Context, user User, sessionData interface{}, responseData interface{}) (*Credential, error) {
	ctx, span := s.tracer.Start(ctx, "webauthn.FinishRegistration", trace.WithAttributes(
		attribute.String("user.name", user.WebAuthnName()),
	))
	defer span.End()

	cred, err := s.next.FinishRegistration(ctx, user, sessionData, responseData)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "webauthn finish registration failed", "error", err)
		return nil, err
	}
	return cred, nil
}

func (s *InstrumentedService) BeginLogin(ctx context.Context, user User) (interface{}, error) {
	ctx, span := s.tracer.Start(ctx, "webauthn.BeginLogin", trace.WithAttributes(
		attribute.String("user.name", user.WebAuthnName()),
	))
	defer span.End()

	res, err := s.next.BeginLogin(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "webauthn begin login failed", "error", err)
		return nil, err
	}
	return res, nil
}

func (s *InstrumentedService) FinishLogin(ctx context.Context, user User, sessionData interface{}, responseData interface{}) (*Credential, error) {
	ctx, span := s.tracer.Start(ctx, "webauthn.FinishLogin", trace.WithAttributes(
		attribute.String("user.name", user.WebAuthnName()),
	))
	defer span.End()

	cred, err := s.next.FinishLogin(ctx, user, sessionData, responseData)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "webauthn finish login failed", "error", err)
		return nil, err
	}
	return cred, nil
}
