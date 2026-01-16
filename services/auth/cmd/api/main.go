package main

import (
	"log"

	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/server"
	"github.com/chris-alexander-pop/system-design-library/services/auth/internal/handler"
)

type Config struct {
	Server   server.Config
	Logger   logger.Config
	Database database.Config
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log := logger.New(cfg.Logger)

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Error("failed to connect to db", "error", err)
		return
	}

	srv := server.New(cfg.Server, log)

	h := handler.New(db)
	h.Register(srv.Echo())

	if err := srv.Start(); err != nil {
		log.Error("server failed", "error", err)
	}
}
