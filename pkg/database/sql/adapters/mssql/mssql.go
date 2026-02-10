package mssql

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Adapter implements the sql.SQL interface for SQL Server.
type Adapter struct {
	db *gorm.DB
}

// New creates a new SQL Server connection.
func New(cfg sql.Config) (sql.SQL, error) {
	if cfg.Driver != database.DriverSQLServer && cfg.Driver != "mssql" {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for mssql adapter", cfg.Driver), nil)
	}

	// Azure SQL often requires encrypt=true
	encryption := "disable"
	if cfg.SSLMode == "require" || cfg.SSLMode == "true" {
		encryption = "true"
	}

	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s&encrypt=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, encryption)

	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to sqlserver")
	}

	return &Adapter{db: db}, nil
}

// Get returns the primary database connection.
func (a *Adapter) Get(ctx context.Context) *gorm.DB {
	return a.db.WithContext(ctx)
}

// GetShard returns a database connection for the given shard key.
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
