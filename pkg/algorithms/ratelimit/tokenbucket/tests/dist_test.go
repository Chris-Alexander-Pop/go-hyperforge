package tokenbucket_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit/tokenbucket"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
)

func BenchmarkDistLimiter_Allow(b *testing.B) {
	// Initialize the limiter
	store := memory.New()
	limiter := tokenbucket.NewDist(store)
	ctx := context.Background()
	key := "benchmark-user"

	// Reset timer to ignore initialization
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Call Allow repeatedly
		// We use a high limit so it likely returns true, but we don't care about the result for perf testing the allocation overhead
		_, _ = limiter.Allow(ctx, key, 1000000, time.Second)
	}
}
