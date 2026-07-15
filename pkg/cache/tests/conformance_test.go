package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

func TestCache(t *testing.T) {
	c := memory.New()
	defer c.Close()
	runCacheConformance(t, c)
}

func TestBloomCacheConformance(t *testing.T) {
	mem := memory.New()
	c := cache.NewBloomCache(mem, cache.BloomCacheConfig{
		ExpectedElements:  1000,
		FalsePositiveRate: 0.01,
	})
	defer c.Close()
	runCacheConformance(t, c)
}

func runCacheConformance(t *testing.T, c cache.Cache) {
	t.Helper()
	ctx := context.Background()

	t.Run("SetGetDelete", func(t *testing.T) {
		key := "test-key-" + t.Name()
		value := "test-value"

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

		if err := c.Delete(ctx, key); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		err := c.Get(ctx, key, &res)
		if err == nil {
			t.Fatal("Expected error after delete, got nil")
		}
		assertNotFound(t, err)
	})

	t.Run("GetMissingIsNotFound", func(t *testing.T) {
		var res string
		err := c.Get(ctx, "missing-"+t.Name(), &res)
		assertNotFound(t, err)
	})

	t.Run("TTLZeroPersists", func(t *testing.T) {
		key := "ttl-zero-" + t.Name()
		if err := c.Set(ctx, key, "forever", 0); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
		var res string
		if err := c.Get(ctx, key, &res); err != nil {
			t.Fatalf("Get after TTL=0 should succeed, got: %v", err)
		}
		if res != "forever" {
			t.Errorf("Expected forever, got %s", res)
		}
	})

	t.Run("ExistsMGetMSetExpireTTL", func(t *testing.T) {
		prefix := "ext-" + t.Name() + ":"
		if err := c.MSet(ctx, map[string]interface{}{
			prefix + "a": "1",
			prefix + "b": "2",
		}, time.Minute); err != nil {
			t.Fatalf("MSet: %v", err)
		}

		ok, err := c.Exists(ctx, prefix+"a")
		if err != nil || !ok {
			t.Fatalf("Exists a: ok=%v err=%v", ok, err)
		}
		ok, err = c.Exists(ctx, prefix+"missing")
		if err != nil || ok {
			t.Fatalf("Exists missing: ok=%v err=%v", ok, err)
		}

		got := map[string]string{}
		if err := c.MGet(ctx, []string{prefix + "a", prefix + "missing", prefix + "b"}, &got); err != nil {
			t.Fatalf("MGet: %v", err)
		}
		if got[prefix+"a"] != "1" || got[prefix+"b"] != "2" {
			t.Fatalf("MGet got %#v", got)
		}
		if _, exists := got[prefix+"missing"]; exists {
			t.Fatal("missing key should be omitted from MGet")
		}

		if err := c.Expire(ctx, prefix+"a", 2*time.Second); err != nil {
			t.Fatalf("Expire: %v", err)
		}
		ttl, err := c.GetTTL(ctx, prefix+"a")
		if err != nil {
			t.Fatalf("GetTTL: %v", err)
		}
		if ttl <= 0 || ttl > 2*time.Second {
			t.Fatalf("unexpected ttl %v", ttl)
		}

		if err := c.Set(ctx, prefix+"persist", "x", 0); err != nil {
			t.Fatal(err)
		}
		ttl, err = c.GetTTL(ctx, prefix+"persist")
		if err != nil {
			t.Fatalf("GetTTL persist: %v", err)
		}
		if ttl != -1 {
			t.Fatalf("expected -1 for no expiry, got %v", ttl)
		}
	})
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	if !cache.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
	if !errors.Is(err, cache.ErrKeyNotFound) && !errors.Is(err, cache.ErrKeyExpired) {
		// Bloom/memory should return the package sentinel for misses.
		var appErr *pkgerrors.AppError
		if !pkgerrors.As(err, &appErr) || appErr.Code != pkgerrors.CodeNotFound {
			t.Fatalf("expected AppError NOT_FOUND via errors.As, got %T %v", err, err)
		}
	}
	var appErr *pkgerrors.AppError
	if !pkgerrors.As(err, &appErr) {
		t.Fatalf("errors.As(*AppError) failed for %v", err)
	}
	if appErr.Code != pkgerrors.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %s", appErr.Code)
	}
}

func TestMemoryTTLZeroPersists(t *testing.T) {
	c := memory.New()
	defer c.Close()
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	var got string
	if err := c.Get(ctx, "k", &got); err != nil {
		t.Fatalf("TTL=0 must not expire immediately: %v", err)
	}
	if got != "v" {
		t.Fatalf("got %q", got)
	}
}

func TestMemoryTTLExpires(t *testing.T) {
	c := memory.New()
	defer c.Close()
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	var got string
	err := c.Get(ctx, "k", &got)
	if !cache.IsNotFound(err) {
		t.Fatalf("expected expired NotFound, got %v", err)
	}
	if !errors.Is(err, cache.ErrKeyExpired) {
		t.Fatalf("expected ErrKeyExpired, got %v", err)
	}
}

func TestResilientCache_NotFoundDoesNotTripBreaker(t *testing.T) {
	c := memory.New()
	defer c.Close()

	rc := cache.NewResilientCache(c, cache.ResilientConfig{
		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 3,
		CircuitBreakerTimeout:   time.Minute,
		RetryEnabled:            true,
		RetryMaxAttempts:        3,
		RetryBackoff:            time.Millisecond,
	})

	ctx := context.Background()
	var dest string
	for i := 0; i < 10; i++ {
		err := rc.Get(ctx, "missing", &dest)
		if !cache.IsNotFound(err) {
			t.Fatalf("iteration %d: expected NotFound, got %v", i, err)
		}
	}

	if state := rc.CircuitBreakerState(); state != resilience.StateClosed {
		t.Fatalf("circuit should stay closed on cache misses, got %s", state)
	}
}

type countingCache struct {
	cache.Cache
	gets atomic.Int64
}

func (c *countingCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.gets.Add(1)
	return c.Cache.Get(ctx, key, dest)
}

func TestResilientCache_NotFoundDoesNotRetry(t *testing.T) {
	inner := &countingCache{Cache: memory.New()}
	defer inner.Close()

	rc := cache.NewResilientCache(inner, cache.ResilientConfig{
		CircuitBreakerEnabled: false,
		RetryEnabled:          true,
		RetryMaxAttempts:      5,
		RetryBackoff:          time.Millisecond,
	})

	var dest string
	err := rc.Get(context.Background(), "missing", &dest)
	if !cache.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
	if n := inner.gets.Load(); n != 1 {
		t.Fatalf("expected 1 Get attempt for NotFound, got %d", n)
	}
}

func TestIsNotFound(t *testing.T) {
	if !cache.IsNotFound(cache.ErrKeyNotFound) {
		t.Fatal("ErrKeyNotFound should be NotFound")
	}
	if !cache.IsNotFound(cache.ErrKeyExpired) {
		t.Fatal("ErrKeyExpired should be NotFound")
	}
	if !cache.IsNotFound(pkgerrors.NotFound("other", nil)) {
		t.Fatal("any CodeNotFound should match")
	}
	if cache.IsNotFound(pkgerrors.Internal("boom", nil)) {
		t.Fatal("Internal should not be NotFound")
	}
	if cache.IsNotFound(nil) {
		t.Fatal("nil should not be NotFound")
	}
}
