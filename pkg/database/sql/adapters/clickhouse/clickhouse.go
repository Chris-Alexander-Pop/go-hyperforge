package clickhouse

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Adapter implements sql.SQL for ClickHouse.
type Adapter struct {
	db *gorm.DB
}

// New creates a ClickHouse connection wrapping gorm.DB as sql.SQL.
func New(cfg sql.Config) (sql.SQL, error) {
	if cfg.Driver != "clickhouse" && cfg.Driver != database.DriverClickHouse {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for clickhouse adapter", cfg.Driver), nil)
	}

	secure := "false"
	if cfg.SSLMode == "require" || cfg.SSLMode == "true" {
		secure = "true"
	}

	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s?secure=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, secure)

	db, err := gorm.Open(clickhouse.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to clickhouse")
	}

	if cfg.MaxOpenConns > 0 || cfg.MaxIdleConns > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get sql.DB")
		}
		if cfg.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		}
		if cfg.MaxOpenConns > 0 {
			sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		}
		if cfg.ConnMaxLifetime > 0 {
			sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
		}
	}

	return &Adapter{db: db}, nil
}

// Get returns the primary database connection.
func (a *Adapter) Get(ctx context.Context) *gorm.DB {
	return a.db.WithContext(ctx)
}

// GetShard ignores key and returns the primary connection.
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

var _ sql.SQL = (*Adapter)(nil)
