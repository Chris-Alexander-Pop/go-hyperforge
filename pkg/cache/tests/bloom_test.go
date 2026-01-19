package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
)

func TestBloomCache(t *testing.T) {
	mem := memory.New()
	cfg := cache.BloomCacheConfig{
		ExpectedElements:  1000,
		FalsePositiveRate: 0.01,
	}

	c := cache.NewBloomCache(mem, cfg)
	defer c.Close()

	ctx := context.Background()
	key := "bloom-key"
	value := "bloom-value"

	// 1. Set (should add to bloom)
	err := c.Set(ctx, key, value, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 2. Get (should pass bloom check)
	var res string
	err = c.Get(ctx, key, &res)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if res != value {
		t.Errorf("Expected %s, got %s", value, res)
	}

	// 3. Negative Lookup (should be blocked by bloom or fail in cache)
	// Since bloom is empty for "missing-key", it should block it early (optimized)
	// or if false positive, fail in cache. Both result in ErrKeyNotFound equivalent (not nil).
	err = c.Get(ctx, "missing-key", &res)
	if err == nil {
		t.Error("Expected error for missing key")
	}
}
