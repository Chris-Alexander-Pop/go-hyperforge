package cache

import (
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// DriverFactory constructs a Cache from root Config.
// Adapters register factories via RegisterDriver (typically from init).
type DriverFactory func(cfg Config) (Cache, error)

var driverFactories = map[string]DriverFactory{}

// RegisterDriver registers a named cache driver factory.
// The memory adapter registers as "memory" on import; redis as "redis".
func RegisterDriver(name string, fn DriverFactory) {
	if name == "" || fn == nil {
		return
	}
	driverFactories[strings.ToLower(name)] = fn
}

// NewFromConfig constructs a Cache for cfg.Driver.
//
// Import adapters to register drivers:
//
//	_ "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
//	_ "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/redis"
//
// Or construct adapters directly via memory.New / redis.New / redis.NewWithClient.
func NewFromConfig(cfg Config) (Cache, error) {
	name := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if name == "" {
		name = "memory"
	}

	fn, ok := driverFactories[name]
	if !ok {
		return nil, errors.InvalidArgument(
			fmt.Sprintf("cache driver %q is not registered; import adapters/memory or adapters/redis", name),
			nil,
		)
	}
	return fn(cfg)
}

// RegisteredDrivers returns the names of drivers currently registered.
func RegisteredDrivers() []string {
	out := make([]string, 0, len(driverFactories))
	for name := range driverFactories {
		out = append(out, name)
	}
	return out
}
