package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

type Config struct {
	Port         string        `env:"PORT" env-default:"8080"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"10s"`
}

type Server struct {
	echo *echo.Echo
	cfg  Config
}

func New(cfg Config) *Server {
	e := echo.New()
	e.HideBanner = true

	// Standard Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())

	// OTel Tracing
	e.Use(otelecho.Middleware("api")) // Service name "api" or configurable

	// Structured Logging (replacing basic console logger)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			// Compute status and error
			status := c.Response().Status
			if err != nil {
				// If Echo returns an error (e.g. 404), status might not be set yet if handler errored early?
				// Using echo.HTTPError logic
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				} else {
					// Generic error, usually 500 equivalent unless handled
					if status == 200 {
						status = 500
					}
				}
			}

			// Log
			logger.L().InfoContext(c.Request().Context(), "http request",
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"status", status,
				"latency", time.Since(start),
				"error", err,
			)
			return err
		}
	})

	// Centralized Error Handler
	e.HTTPErrorHandler = genericErrorHandler

	return &Server{echo: e, cfg: cfg}
}

func (s *Server) Start() error {
	logger.L().InfoContext(context.Background(), "starting http server", "port", s.cfg.Port)
	return s.echo.Start(":" + s.cfg.Port)
}

func (s *Server) Echo() *echo.Echo {
	return s.echo
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// genericErrorHandler maps generic errors (and pkg/errors) to HTTP responses
func genericErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = he.Message.(string)
	} else if e, ok := err.(*errors.AppError); ok {
		// Map Custom Error Codes
		switch e.Code {
		case errors.CodeNotFound:
			code = http.StatusNotFound
		case errors.CodeInvalidArgument:
			code = http.StatusBadRequest
		case errors.CodeUnauthenticated:
			code = http.StatusUnauthorized
		case errors.CodePermissionDenied:
			code = http.StatusForbidden
		}
		msg = e.Message
	}

	// Respond JSON
	_ = c.JSON(code, map[string]interface{}{
		"error": msg,
		"code":  code,
	})
}
