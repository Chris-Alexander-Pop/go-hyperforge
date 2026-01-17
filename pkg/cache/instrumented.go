package cache

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedCache wraps a Cache to add logging and tracing.
type InstrumentedCache struct {
	next   Cache
	tracer trace.Tracer
}

// NewInstrumentedCache creates a new instrumented cache wrapper.
func NewInstrumentedCache(next Cache) *InstrumentedCache {
	return &InstrumentedCache{
		next:   next,
		tracer: otel.Tracer("pkg/cache"),
	}
}

func (c *InstrumentedCache) Get(ctx context.Context, key string, dest interface{}) error {
	ctx, span := c.tracer.Start(ctx, "cache.Get", trace.WithAttributes(
		attribute.String("cache.key", key),
	))
	defer span.End()

	err := c.next.Get(ctx, key, dest)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().DebugContext(ctx, "cache miss", "key", key, "error", err)
		return err
	}

	logger.L().DebugContext(ctx, "cache hit", "key", key)
	return nil
}

func (c *InstrumentedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "cache.Set", trace.WithAttributes(
		attribute.String("cache.key", key),
		attribute.Int64("cache.ttl_ms", ttl.Milliseconds()),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "cache set", "key", key, "ttl", ttl)

	err := c.next.Set(ctx, key, value, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache set failed", "key", key, "error", err)
		return err
	}
	return nil
}

func (c *InstrumentedCache) Delete(ctx context.Context, key string) error {
	ctx, span := c.tracer.Start(ctx, "cache.Delete", trace.WithAttributes(
		attribute.String("cache.key", key),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "cache delete", "key", key)

	err := c.next.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache delete failed", "key", key, "error", err)
		return err
	}
	return nil
}

func (c *InstrumentedCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	ctx, span := c.tracer.Start(ctx, "cache.Incr", trace.WithAttributes(
		attribute.String("cache.key", key),
		attribute.Int64("cache.delta", delta),
	))
	defer span.End()

	val, err := c.next.Incr(ctx, key, delta)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache incr failed", "key", key, "error", err)
		return 0, err
	}

	span.SetAttributes(attribute.Int64("cache.value", val))
	return val, nil
}

func (c *InstrumentedCache) Close() error {
	return c.next.Close()
}
