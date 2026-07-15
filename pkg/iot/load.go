package iot

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
)

// LoadConfig loads iot.Config via pkg/config (env / optional .env) and validates it.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
