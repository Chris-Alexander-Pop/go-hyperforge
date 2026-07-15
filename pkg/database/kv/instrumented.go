package kv

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedKV wraps KV to add logging and tracing.
type InstrumentedKV struct {
	next   KV
	tracer trace.Tracer
}

// NewInstrumentedKV creates a new instrumented KV wrapper.
func NewInstrumentedKV(next KV) *InstrumentedKV {
	return &InstrumentedKV{
		next:   next,
		tracer: otel.Tracer("pkg/database/kv"),
	}
}

// Get retrieves a value by key with tracing.
func (k *InstrumentedKV) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, span := k.tracer.Start(ctx, "kv.Get", trace.WithAttributes(
		attribute.String("kv.key", key),
	))
	defer span.End()

	start := time.Now()
	value, err := k.next.Get(ctx, key)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "kv get failed",
			"key", key,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	logger.L().DebugContext(ctx, "kv get",
		"key", key,
		"size", len(value),
		"duration_ms", duration.Milliseconds(),
	)
	return value, nil
}

// Set stores a value with tracing.
func (k *InstrumentedKV) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, span := k.tracer.Start(ctx, "kv.Set", trace.WithAttributes(
		attribute.String("kv.key", key),
		attribute.Int("kv.value_size", len(value)),
		attribute.Int64("kv.ttl_ms", ttl.Milliseconds()),
	))
	defer span.End()

	start := time.Now()
	err := k.next.Set(ctx, key, value, ttl)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "kv set failed",
			"key", key,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	logger.L().DebugContext(ctx, "kv set",
		"key", key,
		"size", len(value),
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// Delete removes a key with tracing.
func (k *InstrumentedKV) Delete(ctx context.Context, key string) error {
	ctx, span := k.tracer.Start(ctx, "kv.Delete", trace.WithAttributes(
		attribute.String("kv.key", key),
	))
	defer span.End()

	err := k.next.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// Exists checks if a key exists with tracing.
func (k *InstrumentedKV) Exists(ctx context.Context, key string) (bool, error) {
	ctx, span := k.tracer.Start(ctx, "kv.Exists", trace.WithAttributes(
		attribute.String("kv.key", key),
	))
	defer span.End()

	exists, err := k.next.Exists(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return exists, err
}

// Close releases all resources.
func (k *InstrumentedKV) Close() error {
	logger.L().InfoContext(context.Background(), "closing kv database connections")
	return k.next.Close()
}
