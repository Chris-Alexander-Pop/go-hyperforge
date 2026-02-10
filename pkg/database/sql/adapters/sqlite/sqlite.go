package sqlite

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Adapter implements the sql.SQL interface for SQLite.
type Adapter struct {
	db *gorm.DB
}

// New creates a new SQLite connection.
func New(cfg sql.Config) (sql.SQL, error) {
	if cfg.Driver != database.DriverSQLite {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for sqlite adapter", cfg.Driver), nil)
	}

	// For sqlite, Name is used as filepath
	filepath := cfg.Name
	if filepath == "" {
		filepath = "gorm.db"
	}

	db, err := gorm.Open(sqlite.Open(filepath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to sqlite")
	}

	return &Adapter{db: db}, nil
}

// Get returns the primary database connection.
func (a *Adapter) Get(ctx context.Context) *gorm.DB {
	return a.db.WithContext(ctx)
}

// GetShard returns a database connection for the given shard key.
// For single-instance SQLite, this checks if the file is the shard? No, we just return the primary.
func (a *Adapter) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	return a.db.WithContext(ctx), nil
}

// Close releases all database connections.
func (a *Adapter) Close() error {
	sqlDB, err := a.db.DB()
	if err != nil {
		return errors.Wrap(err, "failed to get underlying sql.DB")
	}
	return sqlDB.Close()
}
