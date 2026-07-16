// Package server implements the gateway HTTP API.
package server

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/middleware"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/openapi"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	jwtauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the gateway service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"gateway"`
	Port        string `env:"PORT" env-default:"8080"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`

	JWTSecret string `env:"JWT_SECRET" env-default:"dev-hyperforge-jwt-secret-change-me"`
	JWTIssuer string `env:"JWT_ISSUER" env-default:"go-hyperforge"`

	AuthServiceURL string `env:"AUTH_SERVICE_URL" env-default:"http://127.0.0.1:8081"`
	UserServiceURL string `env:"USER_SERVICE_URL" env-default:"http://127.0.0.1:8082"`
}

// Server wraps the gateway HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
}

// New constructs the gateway HTTP server.
func New(cfg Config, tokens *jwtauth.Adapter) (*Server, error) {
	authURL, err := url.Parse(cfg.AuthServiceURL)
	if err != nil {
		return nil, errors.InvalidArgument("invalid AUTH_SERVICE_URL", err)
	}
	userURL, err := url.Parse(cfg.UserServiceURL)
	if err != nil {
		return nil, errors.InvalidArgument("invalid USER_SERVICE_URL", err)
	}

	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg}

	authProxy := httputil.NewSingleHostReverseProxy(authURL)
	userProxy := newUserProxy(userURL)

	verifier := auth.NewMiddlewareVerifier(tokens)
	authMW := openapi.EchoMiddleware(middleware.AuthMiddleware(verifier))

	e := r.Echo()
	e.GET("/healthz", s.health)

	e.Any("/v1/auth/*", echo.WrapHandler(authProxy))
	e.Any("/v1/auth", echo.WrapHandler(authProxy))

	e.Any("/v1/users/*", echo.WrapHandler(userProxy), authMW)
	e.Any("/v1/users", echo.WrapHandler(userProxy), authMW)

	return s, nil
}

// Echo exposes the underlying Echo instance.
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func newUserProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(req *http.Request) {
		original(req)
		sub := middleware.GetSubject(req.Context())
		roles := middleware.GetRoles(req.Context())
		req.Header.Del("Authorization")
		if sub != "" {
			req.Header.Set("X-User-ID", sub)
		}
		if len(roles) > 0 {
			req.Header.Set("X-User-Roles", strings.Join(roles, ","))
		}
	}
	return proxy
}
