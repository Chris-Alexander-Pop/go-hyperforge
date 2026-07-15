package vector

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedStore wraps Store to add logging and tracing.
type InstrumentedStore struct {
	next   Store
	tracer trace.Tracer
}

// NewInstrumentedStore creates a new instrumented vector store wrapper.
func NewInstrumentedStore(next Store) *InstrumentedStore {
	return &InstrumentedStore{
		next:   next,
		tracer: otel.Tracer("pkg/database/vector"),
	}
}

// Search finds the nearest neighbors with tracing.
func (s *InstrumentedStore) Search(ctx context.Context, vector []float32, limit int) ([]Result, error) {
	ctx, span := s.tracer.Start(ctx, "vector.Search", trace.WithAttributes(
		attribute.Int("vector.dimension", len(vector)),
		attribute.Int("vector.limit", limit),
	))
	defer span.End()

	start := time.Now()
	results, err := s.next.Search(ctx, vector, limit)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "vector search failed",
			"error", err,
			"limit", limit,
			"duration_ms", duration.Milliseconds(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	logger.L().DebugContext(ctx, "vector search completed",
		"results", len(results),
		"limit", limit,
		"duration_ms", duration.Milliseconds(),
	)
	return results, nil
}

// SearchWithOpts finds nearest neighbors with optional metadata filter and tracing.
func (s *InstrumentedStore) SearchWithOpts(ctx context.Context, vector []float32, opts SearchOpts) ([]Result, error) {
	ctx, span := s.tracer.Start(ctx, "vector.SearchWithOpts", trace.WithAttributes(
		attribute.Int("vector.dimension", len(vector)),
		attribute.Int("vector.limit", opts.Limit),
		attribute.Int("vector.filter_keys", len(opts.Filter)),
	))
	defer span.End()

	start := time.Now()
	results, err := s.next.SearchWithOpts(ctx, vector, opts)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "vector search with opts failed",
			"error", err,
			"limit", opts.Limit,
			"duration_ms", duration.Milliseconds(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	logger.L().DebugContext(ctx, "vector search with opts completed",
		"results", len(results),
		"limit", opts.Limit,
		"duration_ms", duration.Milliseconds(),
	)
	return results, nil
}

// Upsert inserts or updates a vector with tracing.
func (s *InstrumentedStore) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	ctx, span := s.tracer.Start(ctx, "vector.Upsert", trace.WithAttributes(
		attribute.String("vector.id", id),
		attribute.Int("vector.dimension", len(vector)),
	))
	defer span.End()

	start := time.Now()
	err := s.next.Upsert(ctx, id, vector, metadata)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "vector upsert failed",
			"id", id,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	logger.L().DebugContext(ctx, "vector upserted",
		"id", id,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// Delete removes a vector with tracing.
func (s *InstrumentedStore) Delete(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "vector.Delete", trace.WithAttributes(
		attribute.String("vector.id", id),
	))
	defer span.End()

	err := s.next.Delete(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// Close releases resources.
func (s *InstrumentedStore) Close() error {
	logger.L().InfoContext(context.Background(), "closing vector store")
	return s.next.Close()
}
