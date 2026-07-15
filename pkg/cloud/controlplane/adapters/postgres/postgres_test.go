package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane"
	cppg "github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane/adapters/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:cp_test_"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestPostgresControlPlane_HostAndInstance(t *testing.T) {
	db := openDB(t)
	cp, err := cppg.New(db, cppg.Config{Dialect: cppg.DialectSQLite})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, cp.Migrate(ctx))

	err = cp.RegisterHost(ctx, cloud.Host{
		ID:       "h1",
		Name:     "node-1",
		Status:   cloud.HostStatusReady,
		Capacity: cloud.Resources{VCPUs: 4, MemoryMB: 8192, DiskGB: 100},
	})
	require.NoError(t, err)

	h, err := cp.GetHost(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, 4, h.Available.VCPUs)

	inst, err := cp.CreateInstance(ctx, controlplane.CreateInstanceRequest{
		Name:      "vm-1",
		HostID:    "h1",
		Resources: cloud.Resources{VCPUs: 2, MemoryMB: 2048, DiskGB: 20},
		Image:     "ubuntu",
	})
	require.NoError(t, err)
	assert.Equal(t, "h1", inst.HostID)

	h, err = cp.GetHost(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, 2, h.Available.VCPUs)

	list, err := cp.ListInstances(ctx, controlplane.ListInstancesOptions{HostID: "h1"})
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, cp.UnbindInstance(ctx, inst.ID))
	h, err = cp.GetHost(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, 4, h.Available.VCPUs)

	require.NoError(t, cp.DeleteInstance(ctx, inst.ID))
	require.NoError(t, cp.DeregisterHost(ctx, "h1"))
}

func TestPostgresControlPlane_DuplicateHost(t *testing.T) {
	db := openDB(t)
	cp, err := cppg.New(db, cppg.Config{Dialect: cppg.DialectSQLite})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, cp.Migrate(ctx))

	host := cloud.Host{ID: "h1", Status: cloud.HostStatusReady, Capacity: cloud.Resources{VCPUs: 1, MemoryMB: 512, DiskGB: 10}}
	require.NoError(t, cp.RegisterHost(ctx, host))
	err = cp.RegisterHost(ctx, host)
	assert.ErrorIs(t, err, controlplane.ErrHostAlreadyRegistered)
}

func TestNew_RequiresDB(t *testing.T) {
	_, err := cppg.New(nil, cppg.Config{})
	require.Error(t, err)
}
