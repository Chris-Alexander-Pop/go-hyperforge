package bigdata_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata"
	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/adapters/memory"
)

func TestBigDataClient(t *testing.T) {
	// Initialize memory adapter
	client := memory.New()

	// Wrap with instrumentation
	instrumented := bigdata.NewInstrumentedClient(client)
	defer instrumented.Close()

	// Execute query
	ctx := context.Background()
	res, err := instrumented.Query(ctx, "SELECT * FROM mock")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Verify results
	if len(res.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(res.Rows))
	}
	if res.Metadata["source"] != "memory" {
		t.Errorf("Expected metadata source 'memory', got %v", res.Metadata["source"])
	}
}
