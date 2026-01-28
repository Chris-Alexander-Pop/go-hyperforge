package ratelimit

// Backward compatibility re-exports from adapters/redis.
// New code should import "github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit/adapters/redis"

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit/adapters/redis"
	goredis "github.com/redis/go-redis/v9"
)

// DistributedLimiter wraps the Redis adapter for backward compatibility.
// Deprecated: Use pkg/api/ratelimit/adapters/redis.New() instead.
type DistributedLimiter = redis.DistributedLimiter

// NewDistributedLimiter creates a new distributed rate limiter.
// Deprecated: Use pkg/api/ratelimit/adapters/redis.New() instead.
func NewDistributedLimiter(client goredis.Cmdable, strategy Strategy) *DistributedLimiter {
	return redis.New(client, redis.Strategy(strategy))
}

// DistributedLimiterInterface for testing/mocking.
type DistributedLimiterInterface interface {
	Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error)
}
