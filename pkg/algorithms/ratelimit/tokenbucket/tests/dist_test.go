package tokenbucket_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit/tokenbucket"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
)

func TestDistLimiter_Allow(t *testing.T) {
	// Create a new limiter with a memory cache backend
	store := memory.New()
	l := tokenbucket.NewDist(store)
	defer store.Close()

	ctx := context.Background()
	key := "user1"
	limit := int64(10)
	period := time.Second

	// Should allow 10 requests immediately (burst capacity)
	for i := 0; i < 10; i++ {
		res, err := l.Allow(ctx, key, limit, period)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// 11th request should fail
	res, _ := l.Allow(ctx, key, limit, period)
	if res.Allowed {
		t.Error("Request 11 should be denied")
	}

	// Wait for refill (10/s => 0.1s for 1 token)
	time.Sleep(110 * time.Millisecond)
	res, _ = l.Allow(ctx, key, limit, period)
	if !res.Allowed {
		t.Error("Request after wait should be allowed")
	}
}

func BenchmarkDistLimiter_Allow(b *testing.B) {
	store := memory.New()
	l := tokenbucket.NewDist(store)
	defer store.Close()

	ctx := context.Background()
	key := "bench-key"
	limit := int64(1000000) // Large limit to avoid hitting rate limit during benchmark
	period := time.Second

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = l.Allow(ctx, key, limit, period)
	}
}

func BenchmarkDistLimiter_Allow_Parallel(b *testing.B) {
	store := memory.New()
	l := tokenbucket.NewDist(store)
	defer store.Close()

	ctx := context.Background()
	key := "bench-key"
	limit := int64(1000000)
	period := time.Second

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = l.Allow(ctx, key, limit, period)
		}
	})
}
