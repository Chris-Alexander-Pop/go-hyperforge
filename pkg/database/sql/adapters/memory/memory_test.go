package memory_test

import (
	"context"
	"testing"

	dbsql "github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorySQL_GetAndPing(t *testing.T) {
	db, err := memory.NewWithConfig(dbsql.Config{Name: "mem_sql_iface"})
	require.NoError(t, err)
	defer db.Close()

	gormDB := db.Get(context.Background())
	require.NotNil(t, gormDB)

	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())
}

func TestMemorySQL_GetShardUnsupported(t *testing.T) {
	db, err := memory.New()
	require.NoError(t, err)
	defer db.Close()

	_, err = db.GetShard(context.Background(), "k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sharding not supported")
}

func TestMemorySQL_ImplementsInterface(t *testing.T) {
	db, err := memory.NewWithConfig(dbsql.Config{Name: "iface"})
	require.NoError(t, err)
	defer db.Close()
	var _ dbsql.SQL = db
}
