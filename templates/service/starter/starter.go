// Package starter is a minimal Hyperforge service bootstrap example.
//
// It shows how to load typed configuration via pkg/config.Load and initialize
// logger + telemetry from that config. Copy this pattern into a real service
// under /services rather than importing this package at runtime.
package starter

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
)

// Config is an example service configuration loaded from env / .env.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"hyperforge-service"`
	HTTPAddr    string `env:"HTTP_ADDR" env-default:":8080"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Load reads Config via pkg/config.Load (process env and optional .env file).
func Load() (Config, error) {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Bootstrap loads config and initializes the process logger via logger.Init.
//
// Example (service main):
//
//	cfg, err := starter.Bootstrap(ctx)
//	if err != nil { log.Fatal(err) }
//	defer func() { _ = logger.Shutdown(context.Background()) }()
//	logger.L().Info("started", "service", cfg.ServiceName)
//
// See also templates/logger for a minimal Init/Shutdown-only helper
// (loggerbootstrap.Bootstrap) when config.Load is not needed yet.
func Bootstrap(ctx context.Context) (Config, error) {
	cfg, err := Load()
	if err != nil {
		return cfg, err
	}
	_ = ctx
	// Required: process-wide slog handler stack (Sampling→Redact→Trace→Async).
	logger.Init(logger.Config{
		Level: cfg.LogLevel,
	})
	return cfg, nil
}
