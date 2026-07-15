package distlock

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedLocker wraps a Locker with logging and tracing.
type InstrumentedLocker struct {
	next   Locker
	tracer trace.Tracer
}

// NewInstrumentedLocker creates a new InstrumentedLocker.
func NewInstrumentedLocker(next Locker) *InstrumentedLocker {
	return &InstrumentedLocker{
		next:   next,
		tracer: otel.Tracer("pkg/concurrency/distlock"),
	}
}

func (l *InstrumentedLocker) NewLock(key string, ttl time.Duration) Lock {
	// We don't trace creation, only lock actions.
	// But we wrap the returned lock.
	lock := l.next.NewLock(key, ttl)
	return &InstrumentedLock{
		next:   lock,
		key:    key,
		tracer: l.tracer,
	}
}

func (l *InstrumentedLocker) Close() error {
	return l.next.Close()
}

// InstrumentedLock wraps a Lock.
type InstrumentedLock struct {
	next   Lock
	key    string
	tracer trace.Tracer
}

func (l *InstrumentedLock) Acquire(ctx context.Context) (bool, error) {
	ctx, span := l.tracer.Start(ctx, "distlock.Acquire", trace.WithAttributes(
		attribute.String("lock.key", l.key),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "acquiring lock", "key", l.key)

	acquired, err := l.next.Acquire(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to acquire lock", "key", l.key, "error", err)
	}
	if acquired {
		span.SetAttributes(attribute.Bool("lock.acquired", true))
		logger.L().DebugContext(ctx, "lock acquired", "key", l.key)
	} else {
		span.SetAttributes(attribute.Bool("lock.acquired", false))
	}
	return acquired, err
}

func (l *InstrumentedLock) Release(ctx context.Context) error {
	ctx, span := l.tracer.Start(ctx, "distlock.Release", trace.WithAttributes(
		attribute.String("lock.key", l.key),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "releasing lock", "key", l.key)

	err := l.next.Release(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to release lock", "key", l.key, "error", err)
	}
	return err
}

func (l *InstrumentedLock) Extend(ctx context.Context, ttl time.Duration) error {
	ctx, span := l.tracer.Start(ctx, "distlock.Extend", trace.WithAttributes(
		attribute.String("lock.key", l.key),
		attribute.String("lock.ttl", ttl.String()),
	))
	defer span.End()

	err := l.next.Extend(ctx, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (l *InstrumentedLock) IsHeld() bool {
	return l.next.IsHeld()
}
