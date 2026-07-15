package fraud

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedDetector wraps a Detector with telemetry.
type InstrumentedDetector struct {
	next   Detector
	tracer trace.Tracer
}

// NewInstrumentedDetector creates a new InstrumentedDetector.
func NewInstrumentedDetector(next Detector) *InstrumentedDetector {
	return &InstrumentedDetector{
		next:   next,
		tracer: otel.Tracer("pkg/security/fraud"),
	}
}

func (d *InstrumentedDetector) Score(ctx context.Context, event UserEvent) (*Evaluation, error) {
	ctx, span := d.tracer.Start(ctx, "Detector.Score",
		trace.WithAttributes(
			attribute.String("user.id", event.UserID),
			attribute.String("event.action", event.Action),
		),
	)
	defer span.End()

	start := time.Now()
	eval, err := d.next.Score(ctx, event)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "fraud check failed",
			"error", err,
			"duration", duration.String(),
			"user_id", event.UserID,
		)
		return nil, err
	}

	span.SetAttributes(
		attribute.Float64("risk.score", eval.RiskScore),
		attribute.String("risk.action", eval.Action),
	)

	logger.L().InfoContext(ctx, "fraud check completed",
		"risk_score", eval.RiskScore,
		"action", eval.Action,
		"duration", duration.String(),
		"user_id", event.UserID,
	)

	return eval, nil
}
