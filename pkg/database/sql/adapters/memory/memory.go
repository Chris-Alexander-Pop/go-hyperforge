package memory

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Adapter implements sql.SQL using an in-memory SQLite database.
// This is the recommended adapter for testing.
type Adapter struct {
	db *gorm.DB
}

// New creates a new in-memory SQLite database.
func New() (sql.SQL, error) {
	return NewWithConfig(sql.Config{})
}

// NewWithConfig creates a new in-memory SQLite database with custom configuration.
func NewWithConfig(cfg sql.Config) (sql.SQL, error) {
	// Use unique in-memory databases per instance
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", cfg.Name)
	if cfg.Name == "" {
		dsn = "file::memory:?cache=shared"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create in-memory sqlite database")
	}

	return &Adapter{db: db}, nil
}

// Get returns the in-memory database connection.
func (a *Adapter) Get(ctx context.Context) *gorm.DB {
	return a.db.WithContext(ctx)
}

// GetShard returns an error as in-memory adapter does not support sharding.
func (a *Adapter) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	return nil, errors.NotFound(fmt.Sprintf("sharding not supported in memory adapter, key: %s", key), nil)
}

// Close closes the in-memory database connection.
func (a *Adapter) Close() error {
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
