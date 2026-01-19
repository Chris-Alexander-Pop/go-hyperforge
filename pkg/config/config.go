// Package config provides environment-based configuration loading and validation.
//
// This package reads configuration from environment variables (and .env files)
// using struct tags, then validates the loaded configuration.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/config"
//
//	type AppConfig struct {
//		Port     int    `env:"PORT" env-default:"8080"`
//		LogLevel string `env:"LOG_LEVEL" env-default:"INFO" validate:"required"`
//	}
//
//	var cfg AppConfig
//	if err := config.Load(&cfg); err != nil {
//		log.Fatal(err)
//	}
package config

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

// Load reads configuration from .env file or environment variables and validates it.
func Load[T any](cfg *T) error {
	// 1. Load from .env if it exists
	if err := cleanenv.ReadConfig(".env", cfg); err != nil {
		// If .env doesn't exist or we just want to rely on env vars,
		// we fallback to ReadEnv to pick up environment variables processing.
		// cleanenv.ReadConfig already does ReadEnv if file fails?
		// Actually cleanenv.ReadConfig returns error if file not found.
		// So we fallback to ReadEnv.
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return errors.Wrap(err, "failed to read env config")
		}
	}

	// 2. Validate the struct
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return errors.Wrap(err, "config validation failed")
	}

	return nil
}
