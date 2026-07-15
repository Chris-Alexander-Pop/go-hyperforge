package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	mpg "github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMeteringPostgres_RecordAndRate(t *testing.T) {
	db, err := sql.Open("sqlite", "file:meter_test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	store, err := mpg.New(db, mpg.Config{Dialect: mpg.DialectSQLite})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, store.Migrate(ctx))

	require.NoError(t, store.SetRate(ctx, metering.RateCard{
		ResourceType: "compute.instance.small",
		PricePerUnit: 0.02,
		Currency:     "USD",
		Unit:         "hour",
	}))

	now := time.Now().UTC()
	require.NoError(t, store.RecordUsage(ctx, metering.UsageEvent{
		TenantID:     "t1",
		ResourceType: "compute.instance.small",
		ResourceID:   "i-1",
		Quantity:     3,
		Timestamp:    now,
		Metadata:     map[string]string{"region": "us"},
	}))

	events, err := store.GetUsage(ctx, metering.UsageFilter{TenantID: "t1"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, 3.0, events[0].Quantity)
	assert.Equal(t, "us", events[0].Metadata["region"])

	cost, err := store.CalculateCost(ctx, events[0])
	require.NoError(t, err)
	assert.InDelta(t, 0.06, cost, 1e-9)

	require.NoError(t, store.Close())
	err = store.RecordUsage(ctx, metering.UsageEvent{
		TenantID: "t1", ResourceType: "x", Quantity: 1,
	})
	require.Error(t, err)
}

func TestMeteringPostgres_RateCRUDHistory(t *testing.T) {
	db, err := sql.Open("sqlite", "file:meter_rate_crud?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	store, err := mpg.New(db, mpg.Config{Dialect: mpg.DialectSQLite})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, store.Migrate(ctx))

	require.NoError(t, store.SetRate(ctx, metering.RateCard{
		ResourceType: "gpu", PricePerUnit: 1, Currency: "USD", Unit: "hour",
	}))
	require.NoError(t, store.UpdateRate(ctx, metering.RateCard{
		ResourceType: "gpu", PricePerUnit: 2, Currency: "USD", Unit: "hour",
	}))
	require.ErrorIs(t, store.UpdateRate(ctx, metering.RateCard{
		ResourceType: "missing", PricePerUnit: 1, Currency: "USD", Unit: "hour",
	}), metering.ErrRateNotFound)

	hist, err := store.ListRateHistory(ctx, "gpu")
	require.NoError(t, err)
	require.Len(t, hist, 2)
	assert.Equal(t, metering.RateOpSet, hist[0].Op)
	assert.Equal(t, metering.RateOpUpdate, hist[1].Op)

	require.NoError(t, store.DeleteRate(ctx, "gpu"))
	_, err = store.GetRate(ctx, "gpu")
	require.ErrorIs(t, err, metering.ErrRateNotFound)
	hist, err = store.ListRateHistory(ctx, "gpu")
	require.NoError(t, err)
	require.Equal(t, metering.RateOpDelete, hist[len(hist)-1].Op)
}

func TestNew_RequiresDB(t *testing.T) {
	_, err := mpg.New(nil, mpg.Config{})
	require.Error(t, err)
}
