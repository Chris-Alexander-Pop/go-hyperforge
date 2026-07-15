/*
Package config provides environment-based configuration loading and validation.

It reads configuration from .env files and process environment variables using
struct tags (via cleanenv), then validates the loaded values through pkg/validator
(including custom tags such as slug, phone_e164, and password_strong).

Load failures return typed pkg/errors.Internal errors. Validation failures return
pkg/errors.InvalidArgument.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/config"

	type AppConfig struct {
		Port     int    `env:"PORT" env-default:"8080"`
		LogLevel string `env:"LOG_LEVEL" env-default:"INFO" validate:"required"`
	}

	var cfg AppConfig
	if err := config.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	// Or load from an explicit file path:
	if err := config.LoadFrom("config.env", &cfg); err != nil {
		log.Fatal(err)
	}
*/
package config
