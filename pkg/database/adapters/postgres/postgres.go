package postgres

import (
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New creates a new Postgres connection using GORM
// It implements database.ConnectionFactory signature
func New(cfg database.Config) (*gorm.DB, error) {
	if cfg.Driver != database.DriverPostgres {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for postgres adapter", cfg.Driver), nil)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// In a real app, use a custom gorm logger that wraps our pkg/logger
		// TBI: Implement GORM-to-pkg/logger adapter
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres")
	}

	// Ensure connection pool settings if needed
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sql.DB")
	}

	// Default pool settings, can be exposed in Config later
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// Register OTel plugin here if needed (e.g., gormotel)
	// if err := db.Use(gormotel.NewPlugin()); err != nil { ... }

	// Note: For pgvector support, ensure the 'vector' type is available in schemas.
	// GORM handles it via raw SQL or by defining a custom data type.
	// Example: db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

	return db, nil
}
