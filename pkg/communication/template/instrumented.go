package template

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// InstrumentedEngine is a wrapper around an Engine that adds observability.
type InstrumentedEngine struct {
	next   Engine
	tracer trace.Tracer
}

// NewInstrumentedEngine creates a new InstrumentedEngine.
func NewInstrumentedEngine(next Engine) *InstrumentedEngine {
	return &InstrumentedEngine{
		next:   next,
		tracer: otel.Tracer("pkg/communication/template"),
	}
}

// Render renders a template with observability.
func (e *InstrumentedEngine) Render(ctx context.Context, templateName string, data interface{}) (string, error) {
	ctx, span := e.tracer.Start(ctx, "template.Render", trace.WithAttributes(
		attribute.String("template.name", templateName),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "rendering template",
		"name", templateName,
	)

	result, err := e.next.Render(ctx, templateName, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to render template",
			"error", err,
			"name", templateName,
		)
	}

	return result, err
}

// Close releases any resources held by the engine.
func (e *InstrumentedEngine) Close() error {
	return e.next.Close()
}
