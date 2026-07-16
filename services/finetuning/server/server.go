// Package server implements the finetuning service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the finetuning service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"finetuning"`
	Port        string `env:"PORT" env-default:"8120"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the finetuning HTTP API.
type Server = crudserver.Server

// New constructs the finetuning HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "finetunes",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "finetunes",
	}, st)
}
