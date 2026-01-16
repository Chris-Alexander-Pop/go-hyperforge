package main

import (
	"log"

	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/server"
	"github.com/chris-alexander-pop/system-design-library/services/user/internal/handler"
)

type Config struct {
	Server  server.Config
	Logger  logger.Config
	NatsURL string `env:"NATS_URL" env-default:"nats://localhost:4222"`
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log := logger.New(cfg.Logger)
	srv := server.New(cfg.Server, log)

	evt, err := events.NewNats(cfg.NatsURL)
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		return
	}
	defer evt.Close()

	h := handler.New(evt)
	h.Register(srv.Echo())

	if err := srv.Start(); err != nil {
		log.Error("server failed", "error", err)
	}
}
