package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/adapters/memory"
)

func TestMemoryAdapter(t *testing.T) {
	adapter := memory.New()
	defer adapter.Close()

	res, err := adapter.Query(context.Background(), "SELECT * FROM users")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(res.Rows) == 0 {
		t.Error("Expected rows")
	}
	if res.Rows[0]["name"] != "mock" {
		t.Error("Expected mock data")
	}
}
