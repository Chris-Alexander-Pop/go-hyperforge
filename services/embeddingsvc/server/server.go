// Package server implements the embeddingsvc service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the embeddingsvc service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"embeddingsvc"`
	Port        string `env:"PORT" env-default:"8099"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the embeddingsvc HTTP API.
type Server = crudserver.Server

// New constructs the embeddingsvc HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "embeddings",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "embeddings",
	}, st)
}
