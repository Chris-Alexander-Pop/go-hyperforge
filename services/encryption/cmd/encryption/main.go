package main

import (
	"os"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/chris-alexander-pop/go-hyperforge/services/encryption/server"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform"
)

func main() {
	var cfg server.Config
	if err := platform.Load(&cfg); err != nil {
		os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(1)
	}
	platform.InitLogger(cfg.LogLevel)

	srv := server.New(cfg)
	logger.L().Info("encryption service starting", "port", cfg.Port, "service", cfg.ServiceName)

	go func() {
		if err := srv.Start(); err != nil {
			logger.L().Error("encryption server stopped", "error", err)
		}
	}()

	if err := platform.WaitForShutdown(10*time.Second, srv.Shutdown); err != nil {
		logger.L().Error("shutdown error", "error", err)
		os.Exit(1)
	}
}
