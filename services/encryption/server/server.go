// Package server implements the encryption service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
	"github.com/labstack/echo/v4"
)

// defaultMasterKeyB64 is a fixed 32-byte AES-256 key (base64) for local/dev.
const defaultMasterKeyB64 = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// Config is the encryption service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"encryption"`
	Port        string `env:"PORT" env-default:"8133"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
	MasterKey   string `env:"ENCRYPTION_MASTER_KEY" env-default:"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="`
}

// Server wraps the encryption HTTP API.
type Server struct {
	rest      *rest.Server
	encryptor *crypto.AESEncryptor
	hasher    *crypto.Hasher
	cfg       Config
}

// New constructs the encryption HTTP server with an in-memory AES key.
func New(cfg Config) *Server {
	key := cfg.MasterKey
	if key == "" {
		key = defaultMasterKeyB64
	}
	enc, err := auth.NewAESEncryptorFromKey(key)
	if err != nil {
		panic("encryption: invalid ENCRYPTION_MASTER_KEY: " + err.Error())
	}
	if enc == nil {
		panic("encryption: ENCRYPTION_MASTER_KEY is required")
	}
	return NewWithEncryptor(cfg, enc)
}

// NewWithEncryptor constructs the server with a custom AES encryptor (tests).
func NewWithEncryptor(cfg Config, encryptor *crypto.AESEncryptor) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:      r,
		encryptor: encryptor,
		hasher:    crypto.NewHasher(crypto.DefaultHashConfig()),
		cfg:       cfg,
	}
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
	e.POST("/v1/encryption/encrypt", s.encrypt)
	e.POST("/v1/encryption/decrypt", s.decrypt)
	e.POST("/v1/encryption/hash", s.hash)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type encryptRequest struct {
	Plaintext string `json:"plaintext"`
}

func (s *Server) encrypt(c echo.Context) error {
	var req encryptRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Plaintext == "" {
		return errors.InvalidArgument("plaintext is required", nil)
	}
	ct, err := s.encryptor.EncryptString(req.Plaintext)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{
		"ciphertext_base64": ct,
	})
}

type decryptRequest struct {
	CiphertextBase64 string `json:"ciphertext_base64"`
}

func (s *Server) decrypt(c echo.Context) error {
	var req decryptRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.CiphertextBase64 == "" {
		return errors.InvalidArgument("ciphertext_base64 is required", nil)
	}
	pt, err := s.encryptor.DecryptString(req.CiphertextBase64)
	if err != nil {
		return errors.InvalidArgument("decryption failed", err)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"plaintext": pt,
	})
}

type hashRequest struct {
	Value string `json:"value"`
}

func (s *Server) hash(c echo.Context) error {
	var req hashRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.Value == "" {
		return errors.InvalidArgument("value is required", nil)
	}
	h, err := s.hasher.Hash(req.Value)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"hash": h})
}
