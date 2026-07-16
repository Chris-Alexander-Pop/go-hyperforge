// Package server implements the cart service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the cart service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"cart"`
	Port        string `env:"PORT" env-default:"8088"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the cart HTTP API.
type Server = crudserver.Server

// New constructs the cart HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "carts",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "carts",
	}, st)
}
