package vision

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedComputerVision wraps ComputerVision with observability.
type InstrumentedComputerVision struct {
	next   ComputerVision
	tracer trace.Tracer
}

// NewInstrumentedComputerVision creates a new InstrumentedComputerVision.
func NewInstrumentedComputerVision(next ComputerVision) *InstrumentedComputerVision {
	return &InstrumentedComputerVision{
		next:   next,
		tracer: otel.Tracer("pkg/ai/perception/vision"),
	}
}

// AnalyzeImage instruments AnalyzeImage.
func (c *InstrumentedComputerVision) AnalyzeImage(ctx context.Context, image Image, features []Feature) (*Analysis, error) {
	ctx, span := c.tracer.Start(ctx, "vision.AnalyzeImage", trace.WithAttributes(
		attribute.String("image.uri", image.URI),
		attribute.Int("features.count", len(features)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "vision analyze image", "uri", image.URI)

	result, err := c.next.AnalyzeImage(ctx, image, features)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "vision analyze image failed", "error", err, "uri", image.URI)
	}

	return result, err
}

// DetectFaces instruments DetectFaces.
func (c *InstrumentedComputerVision) DetectFaces(ctx context.Context, image Image) ([]Face, error) {
	ctx, span := c.tracer.Start(ctx, "vision.DetectFaces", trace.WithAttributes(
		attribute.String("image.uri", image.URI),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "vision detect faces", "uri", image.URI)

	faces, err := c.next.DetectFaces(ctx, image)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "vision detect faces failed", "error", err, "uri", image.URI)
	}

	return faces, err
}
