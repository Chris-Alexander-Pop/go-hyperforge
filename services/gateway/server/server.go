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

	AuthServiceURL         string `env:"AUTH_SERVICE_URL" env-default:"http://127.0.0.1:8081"`
	UserServiceURL         string `env:"USER_SERVICE_URL" env-default:"http://127.0.0.1:8082"`
	PermissionServiceURL   string `env:"PERMISSION_SERVICE_URL" env-default:"http://127.0.0.1:8083"`
	NotificationServiceURL string `env:"NOTIFICATION_SERVICE_URL" env-default:"http://127.0.0.1:8084"`
	EmailServiceURL        string `env:"EMAIL_SERVICE_URL" env-default:"http://127.0.0.1:8085"`
	SMSServiceURL          string `env:"SMS_SERVICE_URL" env-default:"http://127.0.0.1:8086"`
	ProductServiceURL      string `env:"PRODUCT_SERVICE_URL" env-default:"http://127.0.0.1:8087"`
	CartServiceURL         string `env:"CART_SERVICE_URL" env-default:"http://127.0.0.1:8088"`
	OrderServiceURL        string `env:"ORDER_SERVICE_URL" env-default:"http://127.0.0.1:8089"`
	PaymentServiceURL      string `env:"PAYMENT_SERVICE_URL" env-default:"http://127.0.0.1:8090"`
	InventoryServiceURL    string `env:"INVENTORY_SERVICE_URL" env-default:"http://127.0.0.1:8091"`
	AppConfigServiceURL    string `env:"APPCONFIG_SERVICE_URL" env-default:"http://127.0.0.1:8092"`
	AuditServiceURL        string `env:"AUDIT_SERVICE_URL" env-default:"http://127.0.0.1:8093"`
	WorkflowServiceURL     string `env:"WORKFLOW_SERVICE_URL" env-default:"http://127.0.0.1:8094"`
}

type route struct {
	prefix     string
	targetURL  string
	requireJWT bool
	injectUser bool
}

// Server wraps the gateway HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
}

// New constructs the gateway HTTP server.
func New(cfg Config, tokens *jwtauth.Adapter) (*Server, error) {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg}

	verifier := auth.NewMiddlewareVerifier(tokens)
	authMW := openapi.EchoMiddleware(middleware.AuthMiddleware(verifier))

	routes := []route{
		{prefix: "/v1/auth", targetURL: cfg.AuthServiceURL, requireJWT: false},
		{prefix: "/v1/users", targetURL: cfg.UserServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/permissions", targetURL: cfg.PermissionServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/notifications", targetURL: cfg.NotificationServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/emails", targetURL: cfg.EmailServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/sms", targetURL: cfg.SMSServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/products", targetURL: cfg.ProductServiceURL, requireJWT: false},
		{prefix: "/v1/carts", targetURL: cfg.CartServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/orders", targetURL: cfg.OrderServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/payments", targetURL: cfg.PaymentServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/inventory", targetURL: cfg.InventoryServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/configs", targetURL: cfg.AppConfigServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/audits", targetURL: cfg.AuditServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/workflows", targetURL: cfg.WorkflowServiceURL, requireJWT: true, injectUser: true},
	}

	e := r.Echo()
	e.GET("/healthz", s.health)

	for _, rt := range routes {
		proxy, err := newProxy(rt.targetURL, rt.injectUser)
		if err != nil {
			return nil, err
		}
		handler := echo.WrapHandler(proxy)
		if rt.requireJWT {
			e.Any(rt.prefix, handler, authMW)
			e.Any(rt.prefix+"/*", handler, authMW)
		} else {
			e.Any(rt.prefix, handler)
			e.Any(rt.prefix+"/*", handler)
		}
	}

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

func newProxy(rawURL string, injectUser bool) (http.Handler, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.InvalidArgument("invalid upstream URL", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	if !injectUser {
		return proxy, nil
	}
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
	return proxy, nil
}
