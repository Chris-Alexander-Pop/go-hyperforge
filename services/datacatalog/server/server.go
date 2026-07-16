// Package server implements the datacatalog service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the datacatalog service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"datacatalog"`
	Port        string `env:"PORT" env-default:"8140"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the datacatalog HTTP API.
type Server = crudserver.Server

// New constructs the datacatalog HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "catalogs",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "catalogs",
	}, st)
}
