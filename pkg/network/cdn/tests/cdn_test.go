package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/cdn"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/cdn/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestCDNMemoryCreateGetList(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	dist, err := mgr.CreateDistribution(ctx, cdn.CreateDistributionOptions{
		OriginDomain: "origin.example.com",
		Enabled:      true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, dist.ID)
	require.Equal(t, "origin.example.com", dist.Origins[0].DomainName)
	require.True(t, dist.Enabled)

	got, err := mgr.GetDistribution(ctx, dist.ID)
	require.NoError(t, err)
	require.Equal(t, dist.ID, got.ID)

	list, err := mgr.ListDistributions(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestCDNMemoryUpdateEnableDisable(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	dist, err := mgr.CreateDistribution(ctx, cdn.CreateDistributionOptions{
		OriginDomain: "origin.example.com",
		Enabled:      true,
	})
	require.NoError(t, err)

	updated, err := mgr.UpdateDistribution(ctx, dist.ID, cdn.CreateDistributionOptions{
		OriginDomain: "new-origin.example.com",
		Enabled:      true,
	})
	require.NoError(t, err)
	require.Equal(t, "new-origin.example.com", updated.Origins[0].DomainName)

	require.NoError(t, mgr.DisableDistribution(ctx, dist.ID))
	got, err := mgr.GetDistribution(ctx, dist.ID)
	require.NoError(t, err)
	require.False(t, got.Enabled)
	require.Equal(t, cdn.StatusDisabled, got.Status)

	require.NoError(t, mgr.EnableDistribution(ctx, dist.ID))
	got, err = mgr.GetDistribution(ctx, dist.ID)
	require.NoError(t, err)
	require.True(t, got.Enabled)
}

func TestCDNMemoryInvalidationAndDelete(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	dist, err := mgr.CreateDistribution(ctx, cdn.CreateDistributionOptions{
		OriginDomain: "origin.example.com",
		Enabled:      true,
	})
	require.NoError(t, err)

	inv, err := mgr.Invalidate(ctx, dist.ID, []string{"/index.html", "/assets/*"})
	require.NoError(t, err)
	require.NotEmpty(t, inv.ID)

	got, err := mgr.GetInvalidation(ctx, dist.ID, inv.ID)
	require.NoError(t, err)
	require.Equal(t, inv.ID, got.ID)
	require.Len(t, got.Paths, 2)

	_, err = mgr.GetInvalidation(ctx, dist.ID, "missing")
	require.ErrorIs(t, err, cdn.ErrInvalidationNotFound)

	require.NoError(t, mgr.DeleteDistribution(ctx, dist.ID))
	_, err = mgr.GetDistribution(ctx, dist.ID)
	require.ErrorIs(t, err, cdn.ErrDistributionNotFound)
}

func TestCDNMemoryNotFound(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	_, err := mgr.GetDistribution(ctx, "missing")
	require.ErrorIs(t, err, cdn.ErrDistributionNotFound)

	err = mgr.DeleteDistribution(ctx, "missing")
	require.ErrorIs(t, err, cdn.ErrDistributionNotFound)
}
