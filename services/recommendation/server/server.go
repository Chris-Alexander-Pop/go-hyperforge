// Package server implements the recommendation service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the recommendation service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"recommendation"`
	Port        string `env:"PORT" env-default:"8116"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the recommendation HTTP API.
type Server = crudserver.Server

// New constructs the recommendation HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "recommendations",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "recommendations",
	}, st)
}
