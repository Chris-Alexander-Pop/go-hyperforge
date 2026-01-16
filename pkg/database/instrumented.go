package database

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"gorm.io/gorm"
)

// InstrumentedManager wraps Manager to add logging for connection acquisition
type InstrumentedManager struct {
	next DB
}

func NewInstrumentedManager(next DB) *InstrumentedManager {
	return &InstrumentedManager{next: next}
}

func (m *InstrumentedManager) Get(ctx context.Context) *gorm.DB {
	return m.next.Get(ctx)
}

func (m *InstrumentedManager) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	start := time.Now()
	// logger.L().DebugContext(ctx, "resolving shard", "key", key)

	db, err := m.next.GetShard(ctx, key)
	duration := time.Since(start)

	if err != nil {
		logger.L().ErrorContext(ctx, "failed to resolve shard", "key", key, "error", err, "duration", duration)
		return nil, err
	}
	return db, nil
}

func (m *InstrumentedManager) GetDocument(ctx context.Context) interface{} {
	return m.next.GetDocument(ctx)
}

func (m *InstrumentedManager) GetKV(ctx context.Context) interface{} {
	return m.next.GetKV(ctx)
}

func (m *InstrumentedManager) GetVector(ctx context.Context) interface{} {
	return m.next.GetVector(ctx)
}

func (m *InstrumentedManager) Close() error {
	logger.L().Info("closing database connections")
	return m.next.Close()
}
