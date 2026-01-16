package mysql

import (
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New creates a new MySQL connection
func New(cfg database.Config) (*gorm.DB, error) {
	if cfg.Driver != database.DriverMySQL {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for mysql adapter", cfg.Driver), nil)
	}

	// Simplified TLS logic: if SSLMode is "required" or "verify-full", append tls=true (or custom)
	// In a real app we would register a custom TLS config using mysql.RegisterTLSConfig
	// TBI: Implement robust TLS config registration for custom CA usage
	tlsParam := "false"
	if cfg.SSLMode == "require" || cfg.SSLMode == "verify-full" || cfg.SSLMode == "true" {
		tlsParam = "true" // or "custom" if certs are provided
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, tlsParam)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mysql")
	}

	return db, nil
}
