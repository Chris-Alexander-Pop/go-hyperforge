// Package server implements the searchsvc service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the searchsvc service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"searchsvc"`
	Port        string `env:"PORT" env-default:"8109"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the searchsvc HTTP API.
type Server = crudserver.Server

// New constructs the searchsvc HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "search",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "search",
	}, st)
}
