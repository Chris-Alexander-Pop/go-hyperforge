package sql_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sharding"
	dbsql "github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type countingSQL struct {
	inner    dbsql.SQL
	getShard atomic.Int32
	failNext atomic.Bool
}

func (c *countingSQL) Get(ctx context.Context) *gorm.DB {
	return c.inner.Get(ctx)
}

func (c *countingSQL) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	c.getShard.Add(1)
	if c.failNext.Load() {
		c.failNext.Store(false)
		return nil, errors.New("transient shard error")
	}
	return c.inner.Get(ctx), nil
}

func (c *countingSQL) Close() error {
	return c.inner.Close()
}

func TestResilientSQL_ExecuteRetries(t *testing.T) {
	inner, err := memory.NewWithConfig(dbsql.Config{Name: "resilient_exec"})
	require.NoError(t, err)
	defer inner.Close()

	rs := dbsql.NewResilientSQL(inner, dbsql.ResilientConfig{
		CircuitBreakerEnabled: false,
		RetryEnabled:          true,
		RetryMaxAttempts:      3,
		RetryBackoff:          time.Millisecond,
	})

	var calls atomic.Int32
	err = rs.Execute(context.Background(), func(ctx context.Context, db *gorm.DB) error {
		if calls.Add(1) < 2 {
			return errors.New("blip")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestResilientSQL_GetShardRetries(t *testing.T) {
	base, err := memory.NewWithConfig(dbsql.Config{Name: "resilient_shard"})
	require.NoError(t, err)
	defer base.Close()

	inner := &countingSQL{inner: base}
	inner.failNext.Store(true)

	rs := dbsql.NewResilientSQL(inner, dbsql.ResilientConfig{
		CircuitBreakerEnabled: false,
		RetryEnabled:          true,
		RetryMaxAttempts:      3,
		RetryBackoff:          time.Millisecond,
	})

	db, err := rs.GetShard(context.Background(), "k")
	require.NoError(t, err)
	require.NotNil(t, db)
	assert.GreaterOrEqual(t, inner.getShard.Load(), int32(2))
}

func TestResilientSQL_CircuitOpens(t *testing.T) {
	base, err := memory.NewWithConfig(dbsql.Config{Name: "resilient_cb"})
	require.NoError(t, err)
	defer base.Close()

	rs := dbsql.NewResilientSQL(base, dbsql.ResilientConfig{
		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 2,
		CircuitBreakerTimeout:   time.Minute,
		RetryEnabled:            false,
	})

	fail := errors.New("down")
	for i := 0; i < 2; i++ {
		_ = rs.Execute(context.Background(), func(ctx context.Context, db *gorm.DB) error {
			return fail
		})
	}
	assert.Equal(t, resilience.StateOpen, rs.CircuitBreakerState())

	err = rs.Execute(context.Background(), func(ctx context.Context, db *gorm.DB) error {
		return nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, resilience.ErrCircuitOpen)
}

func TestSharded_GetShardRoutes(t *testing.T) {
	a, err := memory.NewWithConfig(dbsql.Config{Name: "shard_a"})
	require.NoError(t, err)
	b, err := memory.NewWithConfig(dbsql.Config{Name: "shard_b"})
	require.NoError(t, err)
	defer a.Close()
	defer b.Close()

	strategy := sharding.NewConsistentHash(50, []string{"a", "b"})
	sharded, err := dbsql.NewSharded(strategy, map[string]dbsql.SQL{
		"a": a,
		"b": b,
	}, "a")
	require.NoError(t, err)

	primary := sharded.Get(context.Background())
	require.NotNil(t, primary)

	db1, err := sharded.GetShard(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotNil(t, db1)

	// Same key must resolve consistently (same underlying shard).
	id1 := strategy.GetShard("user-1")
	id2 := strategy.GetShard("user-1")
	assert.Equal(t, id1, id2)
	db2, err := sharded.GetShard(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotNil(t, db2)

	ids := sharded.ShardIDs()
	assert.ElementsMatch(t, []string{"a", "b"}, ids)
}

func TestSharded_MissingStrategy(t *testing.T) {
	_, err := dbsql.NewSharded(nil, map[string]dbsql.SQL{}, "")
	require.Error(t, err)
}

func TestSharded_UnknownShardMapping(t *testing.T) {
	a, err := memory.NewWithConfig(dbsql.Config{Name: "only_a"})
	require.NoError(t, err)
	defer a.Close()

	// Strategy returns "b" but only "a" is registered → ErrShardNotFound
	strategy := sharding.NewConsistentHash(50, []string{"b"})
	sharded, err := dbsql.NewSharded(strategy, map[string]dbsql.SQL{"a": a}, "a")
	require.NoError(t, err)

	_, err = sharded.GetShard(context.Background(), "anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbsql.ErrShardNotFound)
}
