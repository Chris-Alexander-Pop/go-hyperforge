package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
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

// keyFingerprint returns a non-reversible short hash so logs never contain
// raw cache keys (which may carry session tokens or other sensitive values).
func keyFingerprint(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:8])
}

func (c *InstrumentedCache) Get(ctx context.Context, key string, dest interface{}) error {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.Get", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
	))
	defer span.End()

	err := c.next.Get(ctx, key, dest)
	if err != nil {
		// Cache miss is expected — log at debug, do not mark the span as error.
		if IsNotFound(err) {
			logger.L().DebugContext(ctx, "cache miss", "key_fp", fp)
			return err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache get failed", "key_fp", fp, "error", err)
		return err
	}

	logger.L().DebugContext(ctx, "cache hit", "key_fp", fp)
	return nil
}

func (c *InstrumentedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.Set", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
		attribute.Int64("cache.ttl_ms", ttl.Milliseconds()),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "cache set", "key_fp", fp, "ttl", ttl)

	err := c.next.Set(ctx, key, value, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache set failed", "key_fp", fp, "error", err)
		return err
	}
	return nil
}

func (c *InstrumentedCache) Delete(ctx context.Context, key string) error {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.Delete", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "cache delete", "key_fp", fp)

	err := c.next.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache delete failed", "key_fp", fp, "error", err)
		return err
	}
	return nil
}

func (c *InstrumentedCache) Exists(ctx context.Context, key string) (bool, error) {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.Exists", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
	))
	defer span.End()

	ok, err := c.next.Exists(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	span.SetAttributes(attribute.Bool("cache.exists", ok))
	return ok, nil
}

func (c *InstrumentedCache) MGet(ctx context.Context, keys []string, dest interface{}) error {
	ctx, span := c.tracer.Start(ctx, "cache.MGet", trace.WithAttributes(
		attribute.Int("cache.key_count", len(keys)),
	))
	defer span.End()

	err := c.next.MGet(ctx, keys, dest)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache mget failed", "error", err)
		return err
	}
	return nil
}

func (c *InstrumentedCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "cache.MSet", trace.WithAttributes(
		attribute.Int("cache.key_count", len(items)),
		attribute.Int64("cache.ttl_ms", ttl.Milliseconds()),
	))
	defer span.End()

	err := c.next.MSet(ctx, items, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache mset failed", "error", err)
		return err
	}
	return nil
}

func (c *InstrumentedCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.Expire", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
		attribute.Int64("cache.ttl_ms", ttl.Milliseconds()),
	))
	defer span.End()

	err := c.next.Expire(ctx, key, ttl)
	if err != nil {
		if IsNotFound(err) {
			return err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (c *InstrumentedCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.GetTTL", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
	))
	defer span.End()

	d, err := c.next.GetTTL(ctx, key)
	if err != nil {
		if IsNotFound(err) {
			return 0, err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	span.SetAttributes(attribute.Int64("cache.ttl_ms", d.Milliseconds()))
	return d, nil
}

func (c *InstrumentedCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	fp := keyFingerprint(key)
	ctx, span := c.tracer.Start(ctx, "cache.Incr", trace.WithAttributes(
		attribute.String("cache.key_fp", fp),
		attribute.Int64("cache.delta", delta),
	))
	defer span.End()

	val, err := c.next.Incr(ctx, key, delta)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "cache incr failed", "key_fp", fp, "error", err)
		return 0, err
	}

	span.SetAttributes(attribute.Int64("cache.value", val))
	return val, nil
}

// Unwrap returns the underlying cache.
func (c *InstrumentedCache) Unwrap() Cache {
	return c.next
}

func (c *InstrumentedCache) Close() error {
	return c.next.Close()
}
