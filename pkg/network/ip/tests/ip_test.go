package tests

import (
	"context"
	"net"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/network/ip"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/ip/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestIPMemoryLookupSeeded(t *testing.T) {
	svc := memory.New()
	ctx := context.Background()

	loc, err := svc.Lookup(ctx, "8.8.8.8")
	require.NoError(t, err)
	require.Equal(t, "US", loc.Country)
	require.Equal(t, "Mountain View", loc.City)
	require.Equal(t, 15169, loc.ASN)
}

func TestIPMemoryLookupUnknownAndInvalid(t *testing.T) {
	svc := memory.New()
	ctx := context.Background()

	loc, err := svc.Lookup(ctx, "203.0.113.10")
	require.NoError(t, err)
	require.Equal(t, "XX", loc.Country)
	require.Equal(t, "Unknown", loc.CountryName)

	_, err = svc.Lookup(ctx, "not-an-ip")
	require.ErrorIs(t, err, ip.ErrInvalidIP)
}

func TestIPMemoryBatchThreatBlockCountry(t *testing.T) {
	svc := memory.New()
	ctx := context.Background()

	locs, err := svc.LookupBatch(ctx, []string{"8.8.8.8", "1.1.1.1"})
	require.NoError(t, err)
	require.Len(t, locs, 2)
	require.Equal(t, "US", locs[0].Country)
	require.Equal(t, "AU", locs[1].Country)

	svc.AddThreat("1.2.3.4", &ip.ThreatInfo{
		IP:          net.ParseIP("1.2.3.4"),
		IsThreat:    true,
		ThreatLevel: 80,
		IsVPN:       true,
		Categories:  []string{"vpn"},
	})
	threat, err := svc.GetThreatInfo(ctx, "1.2.3.4")
	require.NoError(t, err)
	require.True(t, threat.IsThreat)
	require.True(t, threat.IsVPN)

	safe, err := svc.GetThreatInfo(ctx, "8.8.8.8")
	require.NoError(t, err)
	require.False(t, safe.IsThreat)

	svc.BlockIP("10.0.0.1")
	blocked, err := svc.IsBlocked(ctx, "10.0.0.1")
	require.NoError(t, err)
	require.True(t, blocked)

	allowed, err := svc.IsCountryAllowed(ctx, "8.8.8.8", []string{"US", "CA"})
	require.NoError(t, err)
	require.True(t, allowed)

	denied, err := svc.IsCountryAllowed(ctx, "1.1.1.1", []string{"US", "CA"})
	require.NoError(t, err)
	require.False(t, denied)
}
