package vector_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridSearch_CombinesKeywordAndVector(t *testing.T) {
	store := memory.New()
	ctx := context.Background()

	require.NoError(t, store.Upsert(ctx, "a", []float32{1, 0}, map[string]interface{}{
		"title": "red apple fruit",
		"tag":   "produce",
	}))
	require.NoError(t, store.Upsert(ctx, "b", []float32{0.9, 0.1}, map[string]interface{}{
		"title": "blue car vehicle",
		"tag":   "auto",
	}))
	require.NoError(t, store.Upsert(ctx, "c", []float32{0.2, 0.8}, map[string]interface{}{
		"title": "apple pie recipe",
		"tag":   "food",
	}))

	results, err := vector.HybridSearch(ctx, store, []float32{1, 0}, vector.HybridOpts{
		Limit:         3,
		KeywordQuery:  "apple",
		KeywordWeight: 0.5,
		VectorWeight:  0.5,
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Both "a" and "c" match keyword; "a" also has strongest vector score.
	assert.Equal(t, "a", results[0].ID)
	assert.Greater(t, results[0].KeywordScore, float32(0))
	assert.Greater(t, results[0].HybridScore, float32(0))

	// Exact title match boosts keyword to 1.
	results2, err := vector.HybridSearch(ctx, store, []float32{0, 1}, vector.HybridOpts{
		Limit:        2,
		KeywordQuery: "apple pie recipe",
	})
	require.NoError(t, err)
	require.NotEmpty(t, results2)
	assert.Equal(t, "c", results2[0].ID)
	assert.Equal(t, float32(1), results2[0].KeywordScore)
}

func TestHybridSearch_FilterAndEmptyQuery(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	require.NoError(t, store.Upsert(ctx, "x", []float32{1, 0}, map[string]interface{}{"cat": "alpha"}))
	require.NoError(t, store.Upsert(ctx, "y", []float32{0.8, 0.2}, map[string]interface{}{"cat": "beta"}))

	results, err := vector.HybridSearch(ctx, store, []float32{1, 0}, vector.HybridOpts{
		Limit:  5,
		Filter: map[string]interface{}{"cat": "alpha"},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "x", results[0].ID)
	assert.Equal(t, float32(1), results[0].KeywordScore) // neutral when no keyword
}
