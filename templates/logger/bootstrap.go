// Package loggerbootstrap is a copy-paste example of process logger Init/Shutdown.
//
// Prefer templates/service/starter for a full config.Load + logger bootstrap.
// This package exists so services can mirror a minimal logger lifecycle without
// inventing their own Init/Shutdown order.
package loggerbootstrap

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
)

// Config is the subset of logger settings typically loaded from env.
type Config struct {
	Level        string  `env:"LOG_LEVEL" env-default:"INFO"`
	Format       string  `env:"LOG_FORMAT" env-default:"JSON"`
	SamplingRate float64 `env:"LOG_SAMPLING_RATE" env-default:"1.0"`
	Async        bool    `env:"LOG_ASYNC" env-default:"true"`
}

// Init configures the process-wide logger. Call Shutdown before exit when Async.
func Init(cfg Config) {
	logger.Init(logger.Config{
		Level:        cfg.Level,
		Format:       cfg.Format,
		SamplingRate: cfg.SamplingRate,
		Async:        cfg.Async,
	})
}

// Shutdown flushes async buffers. Safe to call even if Async was false.
func Shutdown(ctx context.Context) error {
	return logger.Shutdown(ctx)
}

// Bootstrap is Init + a deferred Shutdown helper pattern for main():
//
//	cfg := loggerbootstrap.Config{Level: "INFO"}
//	stop := loggerbootstrap.Bootstrap(cfg)
//	defer stop()
func Bootstrap(cfg Config) (shutdown func()) {
	Init(cfg)
	return func() {
		_ = Shutdown(context.Background())
	}
}
