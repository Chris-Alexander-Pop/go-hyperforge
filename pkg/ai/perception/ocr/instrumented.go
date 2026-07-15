package ocr

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedOCRClient is a wrapper for OCRClient that adds logging and tracing.
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

// DetectText instruments the DetectText method.
func (c *InstrumentedOCRClient) DetectText(ctx context.Context, document Document) (*TextResult, error) {
	ctx, span := c.tracer.Start(ctx, "ocr.DetectText", trace.WithAttributes(
		attribute.String("document.uri", document.URI),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "ocr detect text", "uri", document.URI)

	result, err := c.next.DetectText(ctx, document)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "ocr detect text failed", "error", err, "uri", document.URI)
	}

	return result, err
}
