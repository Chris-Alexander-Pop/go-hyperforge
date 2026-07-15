package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExactStore_Incr(t *testing.T) {
	s := memory.NewExact()
	defer s.Close()
	ctx := context.Background()

	n, err := s.Incr(ctx, "hits", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	n, err = s.Incr(ctx, "hits", 4)
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)

	got, err := s.Count(ctx, "hits")
	require.NoError(t, err)
	assert.Equal(t, int64(5), got)

	missing, err := s.Count(ctx, "nope")
	require.NoError(t, err)
	assert.Equal(t, int64(0), missing)
}

func TestExactStore_AddExact(t *testing.T) {
	s := memory.NewExact()
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.AddExact(ctx, "users", "a"))
	require.NoError(t, s.AddExact(ctx, "users", "b"))
	require.NoError(t, s.AddExact(ctx, "users", "a")) // duplicate

	got, err := s.Count(ctx, "users")
	require.NoError(t, err)
	assert.Equal(t, int64(2), got)

	require.NoError(t, s.Reset(ctx, "users"))
	got, err = s.Count(ctx, "users")
	require.NoError(t, err)
	assert.Equal(t, int64(0), got)
}

func TestExactStore_Closed(t *testing.T) {
	s := memory.NewExact()
	require.NoError(t, s.Close())
	_, err := s.Incr(context.Background(), "x", 1)
	assert.ErrorIs(t, err, analytics.ErrClosed)
}
