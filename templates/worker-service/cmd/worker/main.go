package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	}
	defer evt.Close()

	err = evt.Subscribe(context.Background(), "jobs.created", func(data []byte) error {
		log.Info("received job", "data", string(data))
		return nil
	})
	if err != nil {
		log.Error("failed to subscribe", "error", err)
	}

	log.Info("worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("worker shutting down")
}
