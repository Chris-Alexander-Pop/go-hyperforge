package vector

import (
	"context"
	"sort"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// SearchFunc defines the signature for searching a single shard
type SearchFunc func(ctx context.Context, shardID string, vector []float32, limit int) ([]Result, error)

// NOTE: In a real system you would use a min-heap (priority queue) to maintain top-k across all results efficiently.
// TBI: Implement Min-Heap for aggregation optimization
// For this library, we'll collect all results and sort them, as K is usually small (e.g., 10-100).
// Simplicity vs "Overengineering": We'll implement a Scatter-Gather with timeout.
// TBI: Implement HNSW or IVF index creation and usage for truly large-scale search.

// ScatterGatherSearch executes search across all shards concurrently
func ScatterGatherSearch(
	ctx context.Context,
	vector []float32,
	limit int,
	shardIDs []string,
	searchFn SearchFunc,
) ([]Result, error) {
	// 1. Create a derived context with timeout for "Blazing Fast" requirement
	//    If a shard is slow, we might want to return partial results or fail fast.
	//    Let's assume user passes timeout in ctx, or we enforce a strict SLA here.

	resultsChan := make(chan []Result, len(shardIDs))
	g, ctx := errgroup.WithContext(ctx)

	// 2. Scatter: Launch goroutine for each shard
	for _, id := range shardIDs {
		shardID := id // capture loop var
		g.Go(func() error {
			// Fail-safe: recover from panics in user-provided searchFn
			defer func() {
				if r := recover(); r != nil {
					// Log panic?
				}
			}()

			res, err := searchFn(ctx, shardID, vector, limit)
			if err != nil {
				// We can either fail the whole request or ignore this shard.
				// For "robustness", we might want to log error and assume empty results
				// if partial availability is allowed.
				// Here we'll return error to be strict.
				return errors.Wrap(err, "search failed on shard "+shardID)
			}
			resultsChan <- res
			return nil
		})
	}

	// 3. Wait for completion
	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resultsChan)

	// 4. Gather: Aggregate results
	var allResults []Result
	for res := range resultsChan {
		allResults = append(allResults, res...)
	}

	// 5. Sort: Rerank by Score (Desc for Similarity, Asc for Distance - assuming Similarity here)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// 6. Limit
	if len(allResults) > limit {
		return allResults[:limit], nil
	}

	return allResults, nil
}
