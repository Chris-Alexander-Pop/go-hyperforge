package ratelimit

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
)

// Re-export types for backward compatibility
type Strategy = ratelimit.Strategy
type Result = ratelimit.Result
type Limiter = ratelimit.Limiter

const (
	StrategyFixedWindow   = ratelimit.StrategyFixedWindow
	StrategyTokenBucket   = ratelimit.StrategyTokenBucket
	StrategyLeakyBucket   = ratelimit.StrategyLeakyBucket
	StrategySlidingWindow = ratelimit.StrategySlidingWindow
)

// Factory creates a limiter based on strategy
// Delegates to algorithms package
func New(c cache.Cache, strategy Strategy) Limiter {
	return ratelimit.New(c, strategy)
}
