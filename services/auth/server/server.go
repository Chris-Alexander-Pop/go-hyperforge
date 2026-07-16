// Package server implements the auth service HTTP API.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	jwtauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/chris-alexander-pop/go-hyperforge/services/auth/internal/store"
	"github.com/labstack/echo/v4"
)

// Config is the auth service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"auth"`
	Port        string `env:"PORT" env-default:"8081"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`

	JWTSecret     string        `env:"JWT_SECRET" env-default:"dev-hyperforge-jwt-secret-change-me"`
	JWTIssuer     string        `env:"JWT_ISSUER" env-default:"go-hyperforge"`
	JWTExpiration time.Duration `env:"JWT_EXPIRATION" env-default:"24h"`

	UserServiceURL string `env:"USER_SERVICE_URL" env-default:"http://127.0.0.1:8082"`
}

// Server wraps the auth HTTP API.
type Server struct {
	rest   *rest.Server
	store  *store.Store
	jwt    *jwtauth.Adapter
	cfg    Config
	client *http.Client
}

// New constructs the auth HTTP server with an in-memory credential store.
func New(cfg Config, tokens *jwtauth.Adapter) *Server {
	return NewWithStore(cfg, store.New(), tokens)
}

// NewWithStore constructs the auth HTTP server with a custom store (tests).
func NewWithStore(cfg Config, accounts *store.Store, tokens *jwtauth.Adapter) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:   r,
		store:  accounts,
		jwt:    tokens,
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Second},
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
	e.POST("/v1/auth/register", s.register)
	e.POST("/v1/auth/login", s.login)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}

	acct, err := s.store.Register(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}

	name := req.Name
	if name == "" {
		name = req.Email
	}
	if err := s.createUserProfile(c.Request().Context(), acct.UserID, acct.Email, name); err != nil {
		logger.L().ErrorContext(c.Request().Context(), "failed to create user profile",
			"user_id", acct.UserID, "error", err)
		return errors.Internal("registered credentials but failed to create profile", err)
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"user_id": acct.UserID,
		"email":   acct.Email,
	})
}

func (s *Server) login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}

	acct, err := s.store.Authenticate(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}

	token, err := s.jwt.Generate(acct.UserID, []string{"user"})
	if err != nil {
		return errors.Internal("failed to issue token", err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int64(s.cfg.JWTExpiration.Seconds()),
	})
}

func (s *Server) createUserProfile(ctx context.Context, userID, email, name string) error {
	body, err := json.Marshal(map[string]string{
		"id":    userID,
		"email": email,
		"name":  name,
	})
	if err != nil {
		return err
	}

	url := s.cfg.UserServiceURL + "/v1/users"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return errors.Internal("user service returned unexpected status", nil)
	}
	return nil
}
