package cache

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	cacheredis "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/redis"
	goredis "github.com/redis/go-redis/v9"
)

func TestRedisConformanceMiniredis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c := cacheredis.NewWithClient(client)
	defer c.Close()
	runCacheConformance(t, c)
}

func TestRedisNewFromAddr(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	host, port, err := net.SplitHostPort(mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	c, err := cacheredis.New(cache.Config{Host: host, Port: port})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatal(err)
	}
	ok, err := c.Exists(ctx, "k")
	if err != nil || !ok {
		t.Fatalf("Exists: %v %v", ok, err)
	}
}

func TestNewFromConfigMemory(t *testing.T) {
	// memory adapter registers via init in adapters/memory
	c, err := cache.NewFromConfig(cache.Config{Driver: "memory"})
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	defer c.Close()
	runCacheConformance(t, c)
}

func TestNewFromConfigUnregistered(t *testing.T) {
	_, err := cache.NewFromConfig(cache.Config{Driver: "not-a-driver"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvalidatePrefixRedis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	c := cacheredis.NewWithClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	defer c.Close()
	ctx := context.Background()

	_ = c.MSet(ctx, map[string]interface{}{
		"sess:a": 1,
		"sess:b": 2,
		"keep":   3,
	}, time.Minute)

	n, err := cache.InvalidatePrefix(ctx, cache.NewInstrumentedCache(c), "sess:")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("deleted=%d", n)
	}
}
