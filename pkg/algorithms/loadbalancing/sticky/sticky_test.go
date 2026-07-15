package sticky_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/roundrobin"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/sticky"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSticky_SessionAffinity(t *testing.T) {
	b := sticky.New(nil, "a", "b", "c")
	ctx := context.Background()

	first, err := b.NextKey(ctx, "sess-1")
	require.NoError(t, err)
	for i := 0; i < 20; i++ {
		got, err := b.NextKey(ctx, "sess-1")
		require.NoError(t, err)
		assert.Equal(t, first, got)
	}
	assert.Equal(t, 1, b.AffinitySize())
}

func TestSticky_WithSessionContext(t *testing.T) {
	b := sticky.New(nil, "x", "y")
	ctx := sticky.WithSession(context.Background(), "user-9")
	n1, err := b.Next(ctx)
	require.NoError(t, err)
	n2, err := b.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, n1, n2)
}

func TestSticky_RemapAfterRemove(t *testing.T) {
	b := sticky.New(nil, "a", "b")
	ctx := context.Background()
	node, err := b.NextKey(ctx, "s")
	require.NoError(t, err)

	b.Remove(node)
	next, err := b.NextKey(ctx, "s")
	require.NoError(t, err)
	assert.NotEqual(t, node, next)
	assert.Contains(t, []string{"a", "b"}, next)
}

func TestSticky_Empty(t *testing.T) {
	b := sticky.New(nil)
	_, err := b.Next(context.Background())
	assert.ErrorIs(t, err, loadbalancing.ErrNoNodes)
	b.Add("z", 1)
	n, err := b.NextKey(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "z", n)
}

func TestSticky_Fallback(t *testing.T) {
	inner := roundrobin.New("p", "q")
	b := sticky.New(inner, "p", "q")
	ctx := context.Background()
	n1, err := b.NextKey(ctx, "s1")
	require.NoError(t, err)
	n2, err := b.NextKey(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, n1, n2)
}

func TestSticky_ImplementsBalancer(t *testing.T) {
	var _ loadbalancing.Balancer = sticky.New(nil, "a")
}
