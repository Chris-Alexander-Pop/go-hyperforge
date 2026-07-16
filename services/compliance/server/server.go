// Package server implements the compliance service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the compliance service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"compliance"`
	Port        string `env:"PORT" env-default:"8135"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the compliance HTTP API.
type Server = crudserver.Server

// New constructs the compliance HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "compliance",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "compliance",
	}, st)
}
