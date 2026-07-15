package messaging

import (
	"fmt"
	"strings"
)

// DriverFactory constructs a Broker from root Config.
// Adapters register factories via RegisterDriver (typically from init).
type DriverFactory func(cfg Config) (Broker, error)

var driverFactories = map[string]DriverFactory{}

// RegisterDriver registers a named messaging driver factory.
// The memory adapter registers as "memory" on import.
func RegisterDriver(name string, fn DriverFactory) {
	if name == "" || fn == nil {
		return
	}
	driverFactories[strings.ToLower(name)] = fn
}

// NewFromConfig constructs a Broker for cfg.Driver.
//
// Only the memory driver is registered by default (import
// pkg/messaging/adapters/memory). Other drivers must be constructed via their
// adapter packages to avoid pulling every broker SDK into dependents:
//
//	kafka       → pkg/messaging/adapters/kafka.New
//	rabbitmq    → pkg/messaging/adapters/rabbitmq.New
//	nats        → pkg/messaging/adapters/nats.New
//	sqs         → pkg/messaging/adapters/sqs.New
//	sns         → pkg/messaging/adapters/sns.New
//	gcppubsub   → pkg/messaging/adapters/gcppubsub.New
//	azservicebus→ pkg/messaging/adapters/azservicebus.New
//	redisstreams→ pkg/messaging/adapters/redisstreams.New
//
// Adapter-specific settings (TLS, prefetch, claim-check, credentials, endpoints)
// live on each adapter's Config, not on this root Config.
func NewFromConfig(cfg Config) (Broker, error) {
	name := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if name == "" {
		name = "memory"
	}

	fn, ok := driverFactories[name]
	if !ok {
		return nil, ErrInvalidConfig(
			fmt.Sprintf("driver %q is not registered; import adapters/memory for tests, or construct production brokers via pkg/messaging/adapters/%s", name, name),
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
