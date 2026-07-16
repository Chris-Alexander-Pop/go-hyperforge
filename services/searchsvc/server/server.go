// Package server implements the searchsvc service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	searchmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/data/search/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the searchsvc service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"searchsvc"`
	Port        string `env:"PORT" env-default:"8109"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the search HTTP API.
type Server struct {
	rest   *rest.Server
	engine search.SearchEngine
	cfg    Config
}

// New constructs the searchsvc HTTP server with an in-memory engine.
func New(cfg Config) *Server {
	return NewWithEngine(cfg, searchmemory.New())
}

// NewWithEngine constructs the server with a custom SearchEngine (tests).
func NewWithEngine(cfg Config, engine search.SearchEngine) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, engine: engine, cfg: cfg}
	s.routes()
	return s
}

// Echo exposes the underlying Echo instance (tests / custom mounts).
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/search/indexes/:name/documents", s.indexDocuments)
	e.POST("/v1/search/query", s.query)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type documentInput struct {
	ID   string                 `json:"id"`
	Body map[string]interface{} `json:"document"`
}

type indexDocumentsRequest struct {
	Documents []documentInput        `json:"documents"`
	ID        string                 `json:"id"`
	Document  map[string]interface{} `json:"document"`
}

func (s *Server) indexDocuments(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return errors.InvalidArgument("index name is required", nil)
	}

	var req indexDocumentsRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}

	docs := req.Documents
	if len(docs) == 0 && (req.ID != "" || req.Document != nil) {
		docs = []documentInput{{ID: req.ID, Body: req.Document}}
	}
	if len(docs) == 0 {
		return errors.InvalidArgument("documents are required", nil)
	}

	indexed := make([]string, 0, len(docs))
	for _, d := range docs {
		id := d.ID
		if id == "" {
			id = uuid.NewString()
		}
		body := d.Body
		if body == nil {
			body = map[string]interface{}{}
		}
		if err := s.engine.Index(c.Request().Context(), name, id, body); err != nil {
			return err
		}
		indexed = append(indexed, id)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"index": name,
		"ids":   indexed,
	})
}

type queryRequest struct {
	Index string `json:"index"`
	Query string `json:"query"`
	Size  int    `json:"size,omitempty"`
	From  int    `json:"from,omitempty"`
}

func (s *Server) query(c echo.Context) error {
	var req queryRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Index == "" {
		return errors.InvalidArgument("index is required", nil)
	}
	if req.Query == "" {
		return errors.InvalidArgument("query is required", nil)
	}
	size := req.Size
	if size <= 0 {
		size = 10
	}
	result, err := s.engine.Search(c.Request().Context(), req.Index, search.Query{
		Text: req.Query,
		From: req.From,
		Size: size,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, result)
}
