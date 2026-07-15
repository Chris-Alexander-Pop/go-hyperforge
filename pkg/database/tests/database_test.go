package tests

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql/adapters/postgres"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DatabaseSuite struct {
	*test.Suite
}

func TestDatabaseSuite(t *testing.T) {
	test.Run(t, &DatabaseSuite{Suite: test.NewSuite()})
}

// Mock factory using SQLite for speed
func SqliteFactory(cfg sql.Config) (*gorm.DB, error) {
	// Use in-memory sqlite with unique name based on host/port to simulate distinct shards
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", cfg.Name)
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
}

func (s *DatabaseSuite) TestSQL_Connection() {
	cfg := sql.Config{
		Driver: "sqlite",
		Name:   "primary",
	}

	// Create direct connection
	db, err := SqliteFactory(cfg)
	s.Require().NoError(err)

	sqlDB, err := db.DB()
	s.Require().NoError(err)
	defer sqlDB.Close()

	// Verify connectivity
	s.NoError(sqlDB.Ping())
}

func (s *DatabaseSuite) TestPostgresAdapter_Validation() {
	// Just test that it validates driver
	_, err := postgres.New(sql.Config{Driver: "mysql"})
	s.Error(err)
	s.Contains(err.Error(), "invalid driver")
}
