// Package server implements the mediasvc service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the mediasvc service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"mediasvc"`
	Port        string `env:"PORT" env-default:"8110"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// MediaAsset holds metadata and an optional in-memory blob reference.
type MediaAsset struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int       `json:"size"`
	BlobRef     string    `json:"blob_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Server wraps the media HTTP API.
type Server struct {
	rest   *rest.Server
	cfg    Config
	mu     sync.RWMutex
	assets map[string]MediaAsset
	blobs  map[string][]byte
}

// New constructs the mediasvc HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, assets: make(map[string]MediaAsset), blobs: make(map[string][]byte)}
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
	e.POST("/v1/media", s.create)
	e.GET("/v1/media/:id", s.get)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        string `json:"data,omitempty"` // optional raw string payload stored as blob
}

func (s *Server) create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		return errors.InvalidArgument("filename is required", nil)
	}
	ct := strings.TrimSpace(req.ContentType)
	if ct == "" {
		ct = "application/octet-stream"
	}
	id := uuid.NewString()
	asset := MediaAsset{
		ID:          id,
		Filename:    filename,
		ContentType: ct,
		CreatedAt:   time.Now().UTC(),
	}
	s.mu.Lock()
	if req.Data != "" {
		ref := "blob:" + id
		s.blobs[ref] = []byte(req.Data)
		asset.BlobRef = ref
		asset.Size = len(req.Data)
	}
	s.assets[id] = asset
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, asset)
}

func (s *Server) get(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	s.mu.RLock()
	asset, ok := s.assets[id]
	s.mu.RUnlock()
	if !ok {
		return errors.NotFound("media not found", nil)
	}
	return c.JSON(http.StatusOK, asset)
}
