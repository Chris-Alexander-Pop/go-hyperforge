package document

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedDocument wraps a document.Interface with observability.
type InstrumentedDocument struct {
	next   Interface
	tracer trace.Tracer
}

// NewInstrumented creates a new instrumented wrapper.
func NewInstrumented(next Interface) *InstrumentedDocument {
	return &InstrumentedDocument{
		next:   next,
		tracer: otel.Tracer("pkg/database/document"),
	}
}

func (i *InstrumentedDocument) Insert(ctx context.Context, collection string, doc Document) error {
	ctx, span := i.tracer.Start(ctx, "document.Insert", trace.WithAttributes(
		attribute.String("collection", collection),
	))
	defer span.End()

	err := i.next.Insert(ctx, collection, doc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "document insert failed", "error", err, "collection", collection)
	}
	return err
}

func (i *InstrumentedDocument) Find(ctx context.Context, collection string, query map[string]interface{}) ([]Document, error) {
	ctx, span := i.tracer.Start(ctx, "document.Find", trace.WithAttributes(
		attribute.String("collection", collection),
	))
	defer span.End()

	docs, err := i.next.Find(ctx, collection, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "document find failed", "error", err, "collection", collection)
	} else {
		span.SetAttributes(attribute.Int("result_count", len(docs)))
	}
	return docs, err
}

func (i *InstrumentedDocument) Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	ctx, span := i.tracer.Start(ctx, "document.Update", trace.WithAttributes(
		attribute.String("collection", collection),
	))
	defer span.End()

	err := i.next.Update(ctx, collection, filter, update)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "document update failed", "error", err, "collection", collection)
	}
	return err
}

func (i *InstrumentedDocument) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	ctx, span := i.tracer.Start(ctx, "document.Delete", trace.WithAttributes(
		attribute.String("collection", collection),
	))
	defer span.End()

	err := i.next.Delete(ctx, collection, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "document delete failed", "error", err, "collection", collection)
	}
	return err
}

func (i *InstrumentedDocument) Close() error {
	return i.next.Close()
}
