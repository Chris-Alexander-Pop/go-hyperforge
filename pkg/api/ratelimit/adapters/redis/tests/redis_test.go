package redis_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit/adapters/redis"
	goredis "github.com/redis/go-redis/v9"
)

func TestLimiterInterface(t *testing.T) {
	// Simple test to ensure it compiles
	var _ ratelimit.Limiter = redis.New(&goredis.Client{}, redis.StrategyFixedWindow)
}
