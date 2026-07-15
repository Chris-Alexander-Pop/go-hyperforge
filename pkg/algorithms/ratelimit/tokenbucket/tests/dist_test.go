package tokenbucket_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit/tokenbucket"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistLimiterUsesStore(t *testing.T) {
	store := memory.New()
	defer store.Close()

	a := tokenbucket.NewDist(store)
	b := tokenbucket.NewDist(store)
	ctx := context.Background()
	key := "shared-user"
	limit := int64(3)
	period := time.Minute

	for i := 0; i < 3; i++ {
		res, err := a.Allow(ctx, key, limit, period)
		require.NoError(t, err)
		assert.True(t, res.Allowed, "request %d should be allowed", i)
	}

	// Second limiter instance sharing the same store must observe exhaustion.
	res, err := b.Allow(ctx, key, limit, period)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Equal(t, int64(0), res.Remaining)
}

func BenchmarkDistLimiter_Allow(b *testing.B) {
	store := memory.New()
	limiter := tokenbucket.NewDist(store)
	ctx := context.Background()
	key := "benchmark-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(ctx, key, 1000000, time.Second)
	}
}
