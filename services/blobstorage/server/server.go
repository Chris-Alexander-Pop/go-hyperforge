// Package server implements the blobstorage service HTTP API.
package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	blobmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/memory"
	"github.com/labstack/echo/v4"
)

// Config is the blobstorage service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"blobstorage"`
	Port        string `env:"PORT" env-default:"8145"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the blob storage HTTP API.
type Server struct {
	rest  *rest.Server
	store blob.Store
	cfg   Config
}

// New constructs the blobstorage HTTP server with an in-memory store.
func New(cfg Config) *Server {
	return NewWithStore(cfg, blobmemory.New(blob.Config{}))
}

// NewWithStore constructs the server with a custom blob.Store (tests).
func NewWithStore(cfg Config, store blob.Store) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, store: store, cfg: cfg}
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
	e.POST("/v1/blobs", s.upload)
	e.GET("/v1/blobs/*", s.download)
	e.DELETE("/v1/blobs/*", s.delete)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type uploadRequest struct {
	Key           string `json:"key"`
	ContentBase64 string `json:"content_base64"`
	ContentType   string `json:"content_type,omitempty"`
}

func (s *Server) upload(c echo.Context) error {
	ct := c.Request().Header.Get(echo.HeaderContentType)
	if strings.HasPrefix(ct, "multipart/form-data") {
		return s.uploadMultipart(c)
	}

	var req uploadRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	if req.ContentBase64 == "" {
		return errors.InvalidArgument("content_base64 is required", nil)
	}
	raw, err := base64.StdEncoding.DecodeString(req.ContentBase64)
	if err != nil {
		return errors.InvalidArgument("invalid content_base64", err)
	}
	if err := s.store.Upload(c.Request().Context(), req.Key, bytes.NewReader(raw)); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"key": req.Key,
		"url": s.store.URL(req.Key),
	})
}

func (s *Server) uploadMultipart(c echo.Context) error {
	key := c.FormValue("key")
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	file, err := c.FormFile("file")
	if err != nil {
		return errors.InvalidArgument("file is required", err)
	}
	f, err := file.Open()
	if err != nil {
		return errors.Internal("failed to open upload", err)
	}
	defer f.Close()
	if err := s.store.Upload(c.Request().Context(), key, f); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"key": key,
		"url": s.store.URL(key),
	})
}

func blobKey(c echo.Context) (string, error) {
	key := c.Param("*")
	if key == "" {
		key = c.Param("key")
	}
	if key == "" {
		return "", errors.InvalidArgument("key is required", nil)
	}
	if unescaped, err := url.PathUnescape(key); err == nil {
		key = unescaped
	}
	return key, nil
}

func (s *Server) download(c echo.Context) error {
	key, err := blobKey(c)
	if err != nil {
		return err
	}
	rc, err := s.store.Download(c.Request().Context(), key)
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return errors.Internal("failed to read blob", err)
	}
	if c.QueryParam("encoding") == "raw" {
		return c.Blob(http.StatusOK, "application/octet-stream", data)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"key":            key,
		"content_base64": base64.StdEncoding.EncodeToString(data),
	})
}

func (s *Server) delete(c echo.Context) error {
	key, err := blobKey(c)
	if err != nil {
		return err
	}
	if err := s.store.Delete(c.Request().Context(), key); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
