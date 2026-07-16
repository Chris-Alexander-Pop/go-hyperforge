package main

import (
	"os"
	"time"

	jwtauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/chris-alexander-pop/go-hyperforge/services/auth/server"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform"
)

func main() {
	var cfg server.Config
	if err := platform.Load(&cfg); err != nil {
		os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(1)
	}
	platform.InitLogger(cfg.LogLevel)

	tokens := jwtauth.New(jwtauth.Config{
		Secret:     cfg.JWTSecret,
		Issuer:     cfg.JWTIssuer,
		Expiration: cfg.JWTExpiration,
	})
	srv := server.New(cfg, tokens)

	logger.L().Info("auth service starting", "port", cfg.Port, "service", cfg.ServiceName)

	go func() {
		if err := srv.Start(); err != nil {
			logger.L().Error("auth server stopped", "error", err)
		}
	}()

	if err := platform.WaitForShutdown(10*time.Second, srv.Shutdown); err != nil {
		logger.L().Error("shutdown error", "error", err)
		os.Exit(1)
	}
}
