package captcha

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedVerifier wraps a Verifier with telemetry.
type InstrumentedVerifier struct {
	next   Verifier
	tracer trace.Tracer
}

// NewInstrumentedVerifier creates a new InstrumentedVerifier.
func NewInstrumentedVerifier(next Verifier) *InstrumentedVerifier {
	return &InstrumentedVerifier{
		next:   next,
		tracer: otel.Tracer("pkg/security/captcha"),
	}
}

func (v *InstrumentedVerifier) Verify(ctx context.Context, token string) error {
	ctx, span := v.tracer.Start(ctx, "Verifier.Verify")
	defer span.End()

	start := time.Now()
	err := v.next.Verify(ctx, token)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "captcha verification failed",
			"error", err,
			"duration", duration.String(),
		)
		return err
	}

	logger.L().InfoContext(ctx, "captcha verification passed",
		"duration", duration.String(),
	)

	return nil
}
