package main

import (
	"log"
	"net/url"

	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/server"
	"github.com/labstack/echo/v4/middleware"
)

type Config struct {
	Server     server.Config
	Logger     logger.Config
	AuthSvcURL string `env:"AUTH_SERVICE_URL" env-default:"http://localhost:8081"`
	UserSvcURL string `env:"USER_SERVICE_URL" env-default:"http://localhost:8082"`
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log := logger.New(cfg.Logger)
	srv := server.New(cfg.Server, log)
	e := srv.Echo()

	authURL, _ := url.Parse(cfg.AuthSvcURL)
	userURL, _ := url.Parse(cfg.UserSvcURL)

	targets := []*middleware.ProxyTarget{
		{
			Name: "auth",
			URL:  authURL,
		},
		{
			Name: "user",
			URL:  userURL,
		},
	}

	e.Group("/auth").Use(middleware.Proxy(middleware.NewRoundRobinBalancer(targets[:1])))
	e.Group("/users").Use(middleware.Proxy(middleware.NewRoundRobinBalancer(targets[1:])))

	if err := srv.Start(); err != nil {
		log.Error("gateway failed", "error", err)
	}
}
