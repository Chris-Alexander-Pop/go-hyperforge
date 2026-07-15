package config

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/ilyakaznacheev/cleanenv"
)

// Load reads configuration from a local ".env" file when present, otherwise from
// process environment variables, then validates the result with pkg/validator.
//
// Load failures are returned as errors.Internal. Validation failures are returned
// as errors.InvalidArgument.
func Load[T any](cfg *T) error {
	if err := cleanenv.ReadConfig(".env", cfg); err != nil {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return errors.Internal("failed to read env config", err)
		}
	}
	return validateConfig(cfg)
}

// LoadFrom reads configuration from the file at path (and environment overrides
// via cleanenv), then validates the result with pkg/validator.
//
// Unlike Load, a missing or unreadable file is an error (errors.Internal).
// Validation failures are returned as errors.InvalidArgument.
func LoadFrom[T any](path string, cfg *T) error {
	if path == "" {
		return errors.InvalidArgument("config path is required", nil)
	}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return errors.Internal("failed to read config", err)
	}
	return validateConfig(cfg)
}

func validateConfig[T any](cfg *T) error {
	v := validator.New()
	if err := v.ValidateStruct(cfg); err != nil {
		return errors.InvalidArgument("config validation failed", err)
	}
	return nil
}
