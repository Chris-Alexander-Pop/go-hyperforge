package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/adapters/postgres"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
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
func SqliteFactory(cfg database.Config) (interface{}, error) {
	// Use in-memory sqlite with unique name based on host/port to simulate distinct shards
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", cfg.Name)
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
}

func (s *DatabaseSuite) TestManager_Sharding() {
	cfg := database.ManagerConfig{
		Primary: database.Config{Name: "primary", Driver: "sqlite"},
		Shards: map[string]database.Config{
			"shard-1": {Name: "shard1", Driver: "sqlite"},
			"shard-2": {Name: "shard2", Driver: "sqlite"},
		},
	}

	// create manager with sqlite factory
	mgr, err := database.NewManager(cfg, SqliteFactory)
	s.Require().NoError(err)
	defer mgr.Close()

	// Wrap with instrumentation
	db := database.NewInstrumentedManager(mgr)

	ctx := context.Background()

	// 1. Get Primary
	primary := db.Get(ctx)
	s.NotNil(primary)

	// Verify it's the right DB (sqlite doesn't make it easy to check DSN from *gorm.DB easily without reflection or raw sql,
	// but connectivity check is enough)
	sqlDB, _ := primary.DB()
	s.NoError(sqlDB.Ping())

	// 2. Get Shard 1
	shard1, err := db.GetShard(ctx, "shard-1")
	s.NoError(err)
	s.NotNil(shard1)

	// 3. Get Invalid Shard
	_, err = db.GetShard(ctx, "shard-99")
	s.Error(err)
}

func (s *DatabaseSuite) TestPostgresAdapter_Validation() {
	// Just test that it validates driver
	_, err := postgres.New(database.Config{Driver: "mysql"})
	s.Error(err)
	s.Contains(err.Error(), "invalid driver")
}
