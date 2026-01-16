package mysql

import (
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New creates a new MySQL connection
func New(cfg database.Config) (*gorm.DB, error) {
	if cfg.Driver != database.DriverMySQL {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for mysql adapter", cfg.Driver), nil)
	}

	tlsParam := "false"

	// Load TLS Config Generic
	tlsConfig, err := database.LoadTLSConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load tls config")
	}

	if tlsConfig != nil {
		// Register the custom config with a unique name (hash or just "custom")
		// For simplicity, we use "custom" but in a real system we might need uniqueness if multiple DBs have different certs.
		// Re-registering with the same name is allowed by the driver (overwrites).
		err = mysqldriver.RegisterTLSConfig("custom", tlsConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to register mysql tls config")
		}
		tlsParam = "custom"
	} else if cfg.SSLMode == "require" || cfg.SSLMode == "true" {
		// Fallback to simpler 'true' if no custom certs but SSL wanted (system roots)
		tlsParam = "true"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, tlsParam)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: database.NewGORMLogger().LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mysql")
	}

	return db, nil
}
