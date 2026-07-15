package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchWithOpts_MetadataFilter(t *testing.T) {
	ctx := context.Background()
	store := memory.New()

	require.NoError(t, store.Upsert(ctx, "a", []float32{1, 0}, map[string]interface{}{"lang": "en", "tier": 1}))
	require.NoError(t, store.Upsert(ctx, "b", []float32{0.9, 0.1}, map[string]interface{}{"lang": "fr", "tier": 1}))
	require.NoError(t, store.Upsert(ctx, "c", []float32{0.8, 0.2}, map[string]interface{}{"lang": "en", "tier": 2}))

	results, err := store.SearchWithOpts(ctx, []float32{1, 0}, vector.SearchOpts{
		Limit:  10,
		Filter: map[string]interface{}{"lang": "en"},
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "en", r.Metadata["lang"])
	}

	results, err = store.SearchWithOpts(ctx, []float32{1, 0}, vector.SearchOpts{
		Limit:  10,
		Filter: map[string]interface{}{"lang": "en", "tier": 2},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "c", results[0].ID)
}
