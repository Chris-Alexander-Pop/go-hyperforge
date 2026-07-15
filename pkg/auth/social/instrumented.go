package social

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
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
		tracer: otel.Tracer("pkg/auth/social"),
	}
}

func (p *InstrumentedProvider) GetLoginURL(state string, opts ...oauth2.AuthCodeOption) string {
	// Pure URL generation usually doesn't need heavy tracing, but we can log intent.
	// We can't trace context here easily as it's not passed, but standards say context-first.
	// The interface GetLoginURL(state string, ...) violates strict context-first if strictly I/O,
	// but it's usually pure string gen.
	// We'll skip tracing span start if no context, but we can't really.
	// We'll just pass through.
	return p.next.GetLoginURL(state, opts...)
}

func (p *InstrumentedProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	ctx, span := p.tracer.Start(ctx, "social.Exchange", trace.WithAttributes(
		attribute.String("code", "***"), // Redact code
	))
	defer span.End()

	logger.L().InfoContext(ctx, "exchanging oauth code")

	info, err := p.next.Exchange(ctx, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to exchange oauth code", "error", err)
	}
	return info, err
}
