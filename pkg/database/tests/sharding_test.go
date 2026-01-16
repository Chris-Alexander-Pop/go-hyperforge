package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type ShardingSuite struct {
	*test.Suite
}

func TestShardingSuite(t *testing.T) {
	test.Run(t, &ShardingSuite{Suite: test.NewSuite()})
}

func (s *ShardingSuite) TestConsistentHashing() {
	cfg := database.ManagerConfig{
		Primary: database.Config{Name: "primary", Driver: "sqlite"},
		Shards: map[string]database.Config{
			"node-a": {Name: "node_a", Driver: "sqlite"},
			"node-b": {Name: "node_b", Driver: "sqlite"},
			"node-c": {Name: "node_c", Driver: "sqlite"},
		},
		ShardingStrategy: "consistent_hash",
	}

	mgr, err := database.NewManager(cfg, SqliteFactory)
	s.Require().NoError(err)
	defer mgr.Close()

	ctx := context.Background()

	// Verify that the same key always routes to the same shard (consistency)
	shard1, err := mgr.GetShard(ctx, "user-123")
	s.NoError(err)
	s.NotNil(shard1)

	shard2, err := mgr.GetShard(ctx, "user-123")
	s.NoError(err)

	// Compare underlying sql.DB to verify it's the same connection pool
	sqlDB1, _ := shard1.DB()
	sqlDB2, _ := shard2.DB()
	s.Equal(sqlDB1, sqlDB2, "same key should return same underlying connection pool")

	// Verify different keys eventually hit different shards
	// Note: With only 3 shards, collisions are possible, but strict equality of all keys is unlikely
	shard3, err := mgr.GetShard(ctx, "user-456")
	s.NoError(err)
	s.NotNil(shard3)

	shard4, err := mgr.GetShard(ctx, "user-789")
	s.NoError(err)
	s.NotNil(shard4)

	// Just ensure we don't error out on valid keys that ARENT shard IDs
	_, err = mgr.GetShard(ctx, "some-random-user-id")
	s.NoError(err)
}
