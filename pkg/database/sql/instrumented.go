package sql

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// InstrumentedSQL wraps SQL to add logging and tracing.
type InstrumentedSQL struct {
	next   SQL
	tracer trace.Tracer
}

// NewInstrumentedSQL creates a new instrumented SQL wrapper.
func NewInstrumentedSQL(next SQL) *InstrumentedSQL {
	return &InstrumentedSQL{
		next:   next,
		tracer: otel.Tracer("pkg/database/sql"),
	}
}

// Get returns the primary database connection with tracing.
func (s *InstrumentedSQL) Get(ctx context.Context) *gorm.DB {
	ctx, span := s.tracer.Start(ctx, "sql.Get")
	defer span.End()

	logger.L().DebugContext(ctx, "getting primary sql connection")

	return s.next.Get(ctx)
}

// GetShard returns a shard connection with tracing and logging.
func (s *InstrumentedSQL) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	ctx, span := s.tracer.Start(ctx, "sql.GetShard", trace.WithAttributes(
		attribute.String("shard.key", key),
	))
	defer span.End()

	start := time.Now()
	logger.L().DebugContext(ctx, "resolving shard", "key", key)

	db, err := s.next.GetShard(ctx, key)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "failed to resolve shard",
			"key", key,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	logger.L().DebugContext(ctx, "shard resolved",
		"key", key,
		"duration_ms", duration.Milliseconds(),
	)

	return db, nil
}

// Close releases all database connections.
func (s *InstrumentedSQL) Close() error {
	logger.L().InfoContext(context.Background(), "closing sql database connections")
	return s.next.Close()
}
