package vision

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedComputerVision wraps a ComputerVision client with telemetry.
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

func (c *InstrumentedComputerVision) AnalyzeImage(ctx context.Context, image Image, features []Feature) (*Analysis, error) {
	ctx, span := c.tracer.Start(ctx, "ComputerVision.AnalyzeImage",
		trace.WithAttributes(
			attribute.Int("vision.image_size", len(image.Content)),
			attribute.String("vision.image_uri", image.URI),
			attribute.Int("vision.features_count", len(features)),
		),
	)
	defer span.End()

	start := time.Now()
	analysis, err := c.next.AnalyzeImage(ctx, image, features)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "analyze image failed",
			"error", err,
			"duration", duration.String(),
			"uri", image.URI,
		)
		return nil, err
	}

	logger.L().InfoContext(ctx, "analyze image completed",
		"labels_count", len(analysis.Labels),
		"duration", duration.String(),
		"uri", image.URI,
	)

	return analysis, nil
}

func (c *InstrumentedComputerVision) DetectFaces(ctx context.Context, image Image) ([]Face, error) {
	ctx, span := c.tracer.Start(ctx, "ComputerVision.DetectFaces",
		trace.WithAttributes(
			attribute.Int("vision.image_size", len(image.Content)),
			attribute.String("vision.image_uri", image.URI),
		),
	)
	defer span.End()

	start := time.Now()
	faces, err := c.next.DetectFaces(ctx, image)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "detect faces failed",
			"error", err,
			"duration", duration.String(),
			"uri", image.URI,
		)
		return nil, err
	}

	logger.L().InfoContext(ctx, "detect faces completed",
		"faces_count", len(faces),
		"duration", duration.String(),
		"uri", image.URI,
	)

	return faces, nil
}
