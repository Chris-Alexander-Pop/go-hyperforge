package healthaware_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/healthaware"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/roundrobin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthAware_SkipsUnhealthy(t *testing.T) {
	inner := roundrobin.New("a", "b", "c")
	healthy := map[string]bool{"a": false, "b": true, "c": false}
	b := healthaware.New(inner, healthaware.CheckerFunc(func(ctx context.Context, node string) bool {
		return healthy[node]
	}))

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		node, err := b.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, "b", node)
	}
}

func TestHealthAware_AllUnhealthy(t *testing.T) {
	inner := roundrobin.New("a", "b")
	b := healthaware.New(inner, healthaware.CheckerFunc(func(ctx context.Context, node string) bool {
		return false
	}))

	_, err := b.Next(context.Background())
	assert.ErrorIs(t, err, loadbalancing.ErrNoNodes)
}

func TestHealthAware_NilChecker(t *testing.T) {
	inner := roundrobin.New("x", "y")
	b := healthaware.New(inner, nil)

	node, err := b.Next(context.Background())
	require.NoError(t, err)
	assert.Contains(t, []string{"x", "y"}, node)

	b.Add("z", 1)
	b.Remove("x")
	node, err = b.Next(context.Background())
	require.NoError(t, err)
	assert.Contains(t, []string{"y", "z"}, node)
}
