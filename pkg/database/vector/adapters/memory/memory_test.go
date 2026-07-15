package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryVector_UpsertSearchDelete(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	defer store.Close()

	var _ vector.Store = store

	require.NoError(t, store.Upsert(ctx, "a", []float32{1, 0}, map[string]interface{}{"tag": "x"}))
	require.NoError(t, store.Upsert(ctx, "b", []float32{0, 1}, nil))

	results, err := store.Search(ctx, []float32{1, 0}, 2)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "a", results[0].ID)

	require.NoError(t, store.Delete(ctx, "a"))
	err = store.Delete(ctx, "a")
	require.Error(t, err)
}

func TestMemoryVector_EmptySearch(t *testing.T) {
	store := memory.New()
	defer store.Close()

	results, err := store.Search(context.Background(), []float32{1, 0}, 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}
