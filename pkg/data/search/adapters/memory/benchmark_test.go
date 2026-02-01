package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
)

func BenchmarkBulk(b *testing.B) {
	// Create engine
	engine := New()
	ctx := context.Background()
	indexName := "benchmark-index"

	// Create index
	err := engine.CreateIndex(ctx, indexName, &search.IndexMapping{})
	if err != nil {
		b.Fatalf("failed to create index: %v", err)
	}

	// Prepare bulk operations
	batchSize := 1000
	ops := make([]search.BulkOperation, batchSize)
	for i := 0; i < batchSize; i++ {
		ops[i] = search.BulkOperation{
			Action: search.BulkActionIndex,
			ID:     fmt.Sprintf("doc-%d", i),
			Document: map[string]interface{}{
				"title": "Benchmark Document",
				"value": i,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We use a different ID prefix for each iteration to avoid overwriting (though overwriting is also a valid test case,
		// strict insert might differ from update). Here we just overwrite for simplicity as the map grows.
		// Actually, let's just run the bulk. Overwriting is fine.
		_, err := engine.Bulk(ctx, indexName, ops)
		if err != nil {
			b.Fatalf("bulk failed: %v", err)
		}
	}
}
