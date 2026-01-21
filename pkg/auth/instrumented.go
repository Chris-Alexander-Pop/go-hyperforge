package auth

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedVerifier wraps a Verifier with observability.
type InstrumentedVerifier struct {
	next   Verifier
	tracer trace.Tracer
}

// NewInstrumentedVerifier creates a new InstrumentedVerifier.
func NewInstrumentedVerifier(next Verifier) *InstrumentedVerifier {
	return &InstrumentedVerifier{
		next:   next,
		tracer: otel.Tracer("pkg/auth"),
	}
}

// Verify implements Verifier.
func (v *InstrumentedVerifier) Verify(ctx context.Context, token string) (*Claims, error) {
	ctx, span := v.tracer.Start(ctx, "auth.Verify", trace.WithAttributes(
		attribute.Int("token.length", len(token)),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "verifying token")

	claims, err := v.next.Verify(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "token verification failed", "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("auth.subject", claims.Subject),
		attribute.String("auth.role", claims.Role),
	)
	return claims, nil
}

// InstrumentedIdentityProvider wraps an IdentityProvider with observability.
type InstrumentedIdentityProvider struct {
	next   IdentityProvider
	tracer trace.Tracer
}

// NewInstrumentedIdentityProvider creates a new InstrumentedIdentityProvider.
func NewInstrumentedIdentityProvider(next IdentityProvider) *InstrumentedIdentityProvider {
	return &InstrumentedIdentityProvider{
		next:   next,
		tracer: otel.Tracer("pkg/auth"),
	}
}

// Login implements IdentityProvider.
func (p *InstrumentedIdentityProvider) Login(ctx context.Context, username, password string) (*Claims, error) {
	ctx, span := p.tracer.Start(ctx, "auth.Login", trace.WithAttributes(
		attribute.String("user.username", username),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "attempting login", "username", username)

	claims, err := p.next.Login(ctx, username, password)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "login failed", "username", username, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("user.subject", claims.Subject))
	logger.L().InfoContext(ctx, "login successful", "username", username, "subject", claims.Subject)
	return claims, nil
}
