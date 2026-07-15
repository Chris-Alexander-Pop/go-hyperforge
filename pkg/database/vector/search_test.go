package vector_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScatterGatherSearch_TopK(t *testing.T) {
	searchFn := func(ctx context.Context, shardID string, vec []float32, limit int) ([]vector.Result, error) {
		switch shardID {
		case "s1":
			return []vector.Result{{ID: "low", Score: 0.1}, {ID: "mid", Score: 0.5}}, nil
		case "s2":
			return []vector.Result{{ID: "high", Score: 0.9}, {ID: "mid2", Score: 0.6}}, nil
		default:
			return nil, nil
		}
	}

	results, err := vector.ScatterGatherSearch(
		context.Background(),
		[]float32{1, 0},
		2,
		[]string{"s1", "s2"},
		searchFn,
	)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "high", results[0].ID)
	assert.Equal(t, "mid2", results[1].ID)
}

func TestScatterGatherSearch_EmptyShards(t *testing.T) {
	results, err := vector.ScatterGatherSearch(
		context.Background(),
		[]float32{1},
		3,
		nil,
		func(ctx context.Context, shardID string, vector []float32, limit int) ([]vector.Result, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	)
	require.NoError(t, err)
	assert.Empty(t, results)
}
