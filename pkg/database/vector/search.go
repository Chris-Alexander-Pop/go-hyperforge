package vector

import (
	"context"
	"runtime"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// SearchFunc defines the signature for searching a single shard
type SearchFunc func(ctx context.Context, shardID string, vector []float32, limit int) ([]Result, error)

// ResultHeap removed in favor of datastructures/heap

// ScatterGatherSearch executes search across all shards concurrently
// Implements bounded concurrency and Heap-based aggregation.
func ScatterGatherSearch(
	ctx context.Context,
	vector []float32,
	limit int,
	shardIDs []string,
	searchFn SearchFunc,
) ([]Result, error) {
	// 1. Concurrency Control
	// Limit concurrency to NumCPU * 2 to prevent explosion
	maxConcurrency := int64(runtime.NumCPU() * 2)
	sem := semaphore.NewWeighted(maxConcurrency)

	resultsChan := make(chan []Result, len(shardIDs))
	g, ctx := errgroup.WithContext(ctx)

	// 2. Scatter
	for _, id := range shardIDs {
		shardID := id
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		g.Go(func() error {
			defer sem.Release(1)

			// Fail-safe
			defer func() {
				if r := recover(); r != nil {
					// In real app, log metrics.Counter("panic.search", 1)
				}
			}()

			res, err := searchFn(ctx, shardID, vector, limit)
			if err != nil {
				return errors.Wrap(err, "search failed on shard "+shardID)
			}
			resultsChan <- res
			return nil
		})
	}

	// 3. Wait
	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resultsChan)

	// 4. Gather with Min-Heap (Maintain Top-K)
	// We want to keep the HIGHEST scores.
	// We use a Min-Heap of size K. The root is the "lowest of the top K".
	// If a new item is > root, we pop root and push new item.
	h := heap.NewMinHeap[Result]()

	for shardResults := range resultsChan {
		for _, res := range shardResults {
			if h.Size() < limit {
				h.PushItem(res, float64(res.Score))
			} else {
				// Peek at the smallest item in the heap
				_, minScore, _ := h.Peek()

				// If new score is better than the worst of our top-k
				if float64(res.Score) > minScore {
					h.PopItem()
					h.PushItem(res, float64(res.Score))
				}
			}
		}
	}

	// 5. Extract and Sort (Heap logic leaves them unordered mostly)
	// Pop all gives us smallest to largest of the top K.
	finalResults := make([]Result, h.Size())
	for i := h.Size() - 1; i >= 0; i-- {
		val, _, _ := h.PopItem()
		finalResults[i] = val
	}
	// ResultHeap is Min-Heap.
	// Pop element 1: Smallest.
	// Pop element 2: Second smallest.
	// ...
	// Pop element K: Largest.
	// We filled `finalResults` from back to front (i=Len-1 down to 0).
	// So:
	// finalResults[Len-1] = Smallest
	// finalResults[0] = Largest
	// Thus finalResults is sorted Descending (Highest score first). Correct.

	return finalResults, nil
}
