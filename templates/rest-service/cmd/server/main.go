package main

import (
	"log"

	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/server"
	"github.com/chris-alexander-pop/system-design-library/templates/rest-service/internal/api"
)

type Config struct {
	Server server.Config
	Logger logger.Config
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log := logger.New(cfg.Logger)

	srv := server.New(cfg.Server, log)

	api.RegisterRoutes(srv.Echo())

	if err := srv.Start(); err != nil {
		log.Error("server failed", "error", err)
	}
}
