package ocr

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedOCRClient wraps an OCRClient with telemetry.
type InstrumentedOCRClient struct {
	next   OCRClient
	tracer trace.Tracer
}

// NewInstrumentedOCRClient creates a new InstrumentedOCRClient.
func NewInstrumentedOCRClient(next OCRClient) *InstrumentedOCRClient {
	return &InstrumentedOCRClient{
		next:   next,
		tracer: otel.Tracer("pkg/ai/perception/ocr"),
	}
}

func (c *InstrumentedOCRClient) DetectText(ctx context.Context, document Document) (*TextResult, error) {
	ctx, span := c.tracer.Start(ctx, "OCRClient.DetectText",
		trace.WithAttributes(
			attribute.Int("ocr.document_size", len(document.Content)),
			attribute.String("ocr.document_uri", document.URI),
		),
	)
	defer span.End()

	start := time.Now()
	result, err := c.next.DetectText(ctx, document)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "ocr failed",
			"error", err,
			"duration", duration.String(),
			"uri", document.URI,
		)
		return nil, err
	}

	logger.L().InfoContext(ctx, "ocr completed",
		"pages_count", len(result.Pages),
		"text_len", len(result.Text),
		"duration", duration.String(),
		"uri", document.URI,
	)

	return result, nil
}
