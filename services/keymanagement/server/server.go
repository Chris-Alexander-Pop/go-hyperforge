// Package server implements the keymanagement service HTTP API.
package server

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
	kmsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms/adapters/memory"
	"github.com/labstack/echo/v4"
)

// defaultMasterKeyB64 is a fixed 32-byte AES-256 key (base64) for local/dev.
const defaultMasterKeyB64 = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// Config is the keymanagement service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"keymanagement"`
	Port        string `env:"PORT" env-default:"8134"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
	MasterKey   string `env:"KMS_MASTER_KEY" env-default:"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="`
}

// Server wraps the KMS HTTP API.
type Server struct {
	rest    *rest.Server
	manager kms.KeyManager
	cfg     Config
}

// New constructs the keymanagement HTTP server with an in-memory KMS.
func New(cfg Config) *Server {
	key := cfg.MasterKey
	if key == "" {
		key = defaultMasterKeyB64
	}
	mgr, err := kmsmemory.New(key)
	if err != nil {
		panic("keymanagement: invalid KMS_MASTER_KEY: " + err.Error())
	}
	return NewWithManager(cfg, mgr)
}

// NewWithManager constructs the server with a custom KeyManager (tests).
func NewWithManager(cfg Config, manager kms.KeyManager) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, manager: manager, cfg: cfg}
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
	e.POST("/v1/keys/encrypt", s.encrypt)
	e.POST("/v1/keys/decrypt", s.decrypt)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type encryptRequest struct {
	KeyID     string `json:"key_id"`
	Plaintext string `json:"plaintext"`
}

func (s *Server) encrypt(c echo.Context) error {
	var req encryptRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.KeyID == "" {
		return errors.InvalidArgument("key_id is required", nil)
	}
	if req.Plaintext == "" {
		return errors.InvalidArgument("plaintext is required", nil)
	}
	ct, err := s.manager.Encrypt(c.Request().Context(), req.KeyID, []byte(req.Plaintext))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{
		"key_id":            req.KeyID,
		"ciphertext_base64": base64.StdEncoding.EncodeToString(ct),
	})
}

type decryptRequest struct {
	KeyID            string `json:"key_id"`
	CiphertextBase64 string `json:"ciphertext_base64"`
}

func (s *Server) decrypt(c echo.Context) error {
	var req decryptRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.KeyID == "" {
		return errors.InvalidArgument("key_id is required", nil)
	}
	if req.CiphertextBase64 == "" {
		return errors.InvalidArgument("ciphertext_base64 is required", nil)
	}
	raw, err := base64.StdEncoding.DecodeString(req.CiphertextBase64)
	if err != nil {
		return errors.InvalidArgument("invalid ciphertext_base64", err)
	}
	pt, err := s.manager.Decrypt(c.Request().Context(), req.KeyID, raw)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{
		"key_id":    req.KeyID,
		"plaintext": string(pt),
	})
}
