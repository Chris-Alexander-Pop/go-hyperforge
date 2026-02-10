package mysql

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Adapter implements the sql.SQL interface for MySQL.
type Adapter struct {
	db *gorm.DB
}

// New creates a new MySQL connection.
func New(cfg sql.Config) (sql.SQL, error) {
	if cfg.Driver != database.DriverMySQL {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for mysql adapter", cfg.Driver), nil)
	}

	tlsParam := "false"

	// Load TLS Config
	tlsConfig, err := database.LoadTLSConfig(cfg.SSLMode, cfg.SSLRootCert, cfg.SSLCert, cfg.SSLKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load tls config")
	}

	if tlsConfig != nil {
		// We use "custom" as the key for registered TLS config
		err = mysqldriver.RegisterTLSConfig("custom", tlsConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to register mysql tls config")
		}
		tlsParam = "custom"
	} else if cfg.SSLMode == "require" || cfg.SSLMode == "true" {
		tlsParam = "true"
	}

	// Correct DSN format for go-sql-driver/mysql
	// user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local&tls=...
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, tlsParam)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: database.NewGORMLogger().LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mysql")
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
