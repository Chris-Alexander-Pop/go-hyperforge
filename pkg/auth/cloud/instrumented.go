package cloud

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedIdentityProvider wraps an IdentityProvider with logging and tracing.
type InstrumentedIdentityProvider struct {
	next   IdentityProvider
	tracer trace.Tracer
}

// NewInstrumentedIdentityProvider creates a new InstrumentedIdentityProvider.
func NewInstrumentedIdentityProvider(next IdentityProvider) *InstrumentedIdentityProvider {
	return &InstrumentedIdentityProvider{
		next:   next,
		tracer: otel.Tracer("pkg/auth/cloud"),
	}
}

func (p *InstrumentedIdentityProvider) SignUp(ctx context.Context, username, password string, attributes map[string]string) error {
	ctx, span := p.tracer.Start(ctx, "cloud.SignUp", trace.WithAttributes(
		attribute.String("username", username),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "signing up user", "username", username)

	err := p.next.SignUp(ctx, username, password, attributes)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to sign up user", "username", username, "error", err)
	}
	return err
}

func (p *InstrumentedIdentityProvider) SignIn(ctx context.Context, username, password string) (*AuthResult, error) {
	ctx, span := p.tracer.Start(ctx, "cloud.SignIn", trace.WithAttributes(
		attribute.String("username", username),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "signing in user", "username", username)

	res, err := p.next.SignIn(ctx, username, password)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to sign in user", "username", username, "error", err)
	}
	return res, err
}
