package main

import (
	"context"
	"log"

	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

type Config struct {
	Logger  logger.Config
	NatsURL string `env:"NATS_URL" env-default:"nats://localhost:4222"`
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log := logger.New(cfg.Logger)

	evt, err := events.NewNats(cfg.NatsURL)
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		return
	}
	defer evt.Close()

	log.Info("notification worker started")

	err = evt.Subscribe(context.Background(), "user.registered", func(data []byte) error {
		log.Info("sending welcome email", "payload", string(data))
		return nil
	})

	if err != nil {
		log.Error("failed to subscribe", "error", err)
	}

	select {}
}
