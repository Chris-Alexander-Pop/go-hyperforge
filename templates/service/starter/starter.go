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

// Bootstrap loads config and initializes the process logger.
// Callers should defer logger.Shutdown when Async logging is enabled.
// See also templates/logger for a minimal Init/Shutdown bootstrap helper.
func Bootstrap(ctx context.Context) (Config, error) {
	cfg, err := Load()
	if err != nil {
		return cfg, err
	}
	_ = ctx
	logger.Init(logger.Config{
		Level: cfg.LogLevel,
	})
	return cfg, nil
}
