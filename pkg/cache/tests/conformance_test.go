package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
)

func TestCache(t *testing.T) {
	// Use memory adapter
	c := memory.New()
	defer c.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// 1. Set
	err := c.Set(ctx, key, value, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 2. Get
	var res string
	err = c.Get(ctx, key, &res)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if res != value {
		t.Errorf("Expected %s, got %s", value, res)
	}

	// 3. Delete
	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 4. Verify Gone
	err = c.Get(ctx, key, &res)
	if err == nil {
		t.Error("Expected error after delete, got nil")
	}
}
