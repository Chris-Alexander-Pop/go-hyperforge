package ratelimit

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit/fixedwindow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit/leakybucket"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit/slidingwindow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit/tokenbucket"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
)

// Strategy defines the rate limiting algorithm.
type Strategy int

const (
	StrategyTokenBucket Strategy = iota
	StrategyLeakyBucket
	StrategyFixedWindow
	StrategySlidingWindow
)

// Re-export types for backward compatibility
type Result = ratelimit.Result
type Limiter = ratelimit.Limiter

// Factory creates a limiter based on strategy
// Delegates to algorithms package
func New(c cache.Cache, strategy Strategy) Limiter {
	switch strategy {
	case StrategyTokenBucket:
		return tokenbucket.NewDist(c)
	case StrategyLeakyBucket:
		return leakybucket.New(c)
	case StrategyFixedWindow:
		return fixedwindow.New(c)
	case StrategySlidingWindow:
		fallthrough
	default:
		return slidingwindow.New(c)
	}
}
