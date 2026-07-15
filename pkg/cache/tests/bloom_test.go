package cache

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
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

	if err := c.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var res string
	if err := c.Get(ctx, key, &res); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if res != value {
		t.Errorf("Expected %s, got %s", value, res)
	}

	err := c.Get(ctx, "missing-key", &res)
	if err == nil {
		t.Fatal("Expected error for missing key")
	}
	if !cache.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound sentinel, got %v", err)
	}
	var appErr *errors.AppError
	if !errors.As(err, &appErr) || appErr.Code != errors.CodeNotFound {
		t.Fatalf("errors.As CodeNotFound failed: %v", err)
	}
}
