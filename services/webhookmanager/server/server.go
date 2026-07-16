// Package server implements the webhookmanager service HTTP API.
package server

import (
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/crudserver"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

// Config is the webhookmanager service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"webhookmanager"`
	Port        string `env:"PORT" env-default:"8130"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server is the webhookmanager HTTP API.
type Server = crudserver.Server

// New constructs the webhookmanager HTTP server.
func New(cfg Config) *Server {
	return crudserver.New(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "webhooks",
	})
}

// NewWithStore constructs the server with a custom store.
func NewWithStore(cfg Config, st *memstore.Store) *Server {
	return crudserver.NewWithStore(crudserver.Config{
		ServiceName: cfg.ServiceName,
		Port:        cfg.Port,
		Resource:    "webhooks",
	}, st)
}
