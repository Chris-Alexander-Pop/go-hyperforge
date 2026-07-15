package scanning

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedScanner wraps a Scanner with telemetry.
type InstrumentedScanner struct {
	next   Scanner
	tracer trace.Tracer
}

// NewInstrumentedScanner creates a new InstrumentedScanner.
func NewInstrumentedScanner(next Scanner) *InstrumentedScanner {
	return &InstrumentedScanner{
		next:   next,
		tracer: otel.Tracer("pkg/security/scanning"),
	}
}

func (s *InstrumentedScanner) Scan(ctx context.Context, resource Resource) (*Report, error) {
	ctx, span := s.tracer.Start(ctx, "Scanner.Scan",
		trace.WithAttributes(
			attribute.String("resource.id", resource.ID),
			attribute.String("resource.type", resource.Type),
		),
	)
	defer span.End()

	start := time.Now()
	report, err := s.next.Scan(ctx, resource)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "scan failed", "error", err, "resource_id", resource.ID)
		return nil, err
	}

	span.SetAttributes(attribute.Bool("scan.clean", report.Clean))
	logger.L().InfoContext(ctx, "scan completed",
		"clean", report.Clean,
		"threats", len(report.Threats),
		"duration", duration.String(),
	)

	return report, nil
}
