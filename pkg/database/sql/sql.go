// Package sql provides a unified interface for SQL databases.
//
// Supported backends:
//   - PostgreSQL
//   - MySQL
//   - SQLite
//   - SQL Server (MSSQL)
//   - ClickHouse
//
// Features:
//   - Connection pooling and lifecycle management
//   - Multi-database sharding support
//   - Instrumented wrapper for logging and tracing
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql/adapters/postgres"
//
//	db, err := postgres.New(cfg)
//	gormDB := db.Get(ctx)
package sql

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Config holds configuration for a SQL database connection.
type Config struct {
	// Driver specifies the SQL backend: "postgres", "mysql", "sqlite", "mssql", "clickhouse".
	Driver string `env:"SQL_DRIVER" env-default:"postgres"`

	// Host is the database server hostname.
	Host string `env:"SQL_HOST" env-default:"localhost"`

	// Port is the database server port.
	Port string `env:"SQL_PORT" env-default:"5432"`

	// User is the database username.
	User string `env:"SQL_USER" env-default:"postgres"`

	// Password is the database password.
	Password string `env:"SQL_PASSWORD"`

	// Name is the database name.
	Name string `env:"SQL_NAME" env-default:"postgres"`

	// SSL Configuration
	SSLMode     string `env:"SQL_SSL_MODE" env-default:"disable"` // disable, require, verify-ca, verify-full
	SSLCert     string `env:"SQL_SSL_CERT"`                       // Path to client cert
	SSLKey      string `env:"SQL_SSL_KEY"`                        // Path to client key
	SSLRootCert string `env:"SQL_SSL_ROOT_CERT"`                  // Path to CA cert
	AuthToken   string `env:"SQL_AUTH_TOKEN"`                     // For IAM authentication (e.g. AWS RDS)

	// Connection Pooling
	MaxOpenConns    int           `env:"SQL_MAX_OPEN_CONNS" env-default:"10"`
	MaxIdleConns    int           `env:"SQL_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `env:"SQL_CONN_MAX_LIFETIME" env-default:"5m"`
}

// SQL defines the interface for SQL database operations.
type SQL interface {
	// Get returns the primary database connection.
	Get(ctx context.Context) *gorm.DB

	// GetShard returns a database connection for the given shard key.
	GetShard(ctx context.Context, key string) (*gorm.DB, error)

	// Close releases all database connections.
	Close() error
}
