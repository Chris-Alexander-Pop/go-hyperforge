package api

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
)

// LoadConfig loads api.Config via pkg/config (env / optional .env) and validates it.
// Note: GraphQLSchema cannot be loaded from env and must be set by the caller.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
