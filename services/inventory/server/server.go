// Package server implements the inventory service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the inventory service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"inventory"`
	Port        string `env:"PORT" env-default:"8091"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the inventory HTTP API.
type Server = crudserver.Server

// New constructs the inventory HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "inventory",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "inventory",
	}, st)
}
