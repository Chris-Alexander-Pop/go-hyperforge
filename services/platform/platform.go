// Package platform provides small bootstrap helpers for Hyperforge services.
//
// These helpers are intentionally thin: load typed config, initialize the
// process logger, and shut down cleanly. Copy or call them from service mains;
// do not grow this package into an application framework.
package platform

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
)

// BaseConfig is the common env surface every service should expose.
type BaseConfig struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"hyperforge-service"`
	Port        string `env:"PORT" env-default:"8080"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Load reads configuration via pkg/config.Load.
func Load[T any](cfg *T) error {
	return config.Load(cfg)
}

// InitLogger initializes the process-wide logger.
func InitLogger(level string) {
	if level == "" {
		level = "info"
	}
	logger.Init(logger.Config{Level: level})
}

// Bootstrap loads cfg and initializes the logger with logLevel.
// Prefer Load + InitLogger(cfg.LogLevel) when the config type has a LogLevel field.
func Bootstrap[T any](cfg *T, logLevel string) error {
	if err := Load(cfg); err != nil {
		return err
	}
	InitLogger(logLevel)
	return nil
}

// WaitForShutdown blocks until SIGINT/SIGTERM, then runs shutdown with timeout.
func WaitForShutdown(timeout time.Duration, shutdown func(context.Context) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := shutdown(shutdownCtx); err != nil {
		_ = logger.Shutdown(context.Background())
		return err
	}
	return logger.Shutdown(context.Background())
}
