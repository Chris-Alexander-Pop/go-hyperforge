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

	// Apply configured read/write timeouts to the underlying http.Server.
	if cfg.ReadTimeout > 0 {
		e.Server.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		e.Server.WriteTimeout = cfg.WriteTimeout
	}

	// Standard Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())

	// OTel Tracing
	e.Use(otelecho.Middleware("api"))

	// Structured Logging
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			status := c.Response().Status
			if err != nil {
				var he *echo.HTTPError
				if errors.As(err, &he) {
					status = he.Code
				} else if status == http.StatusOK {
					status = errors.HTTPStatus(err)
				}
			}

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

// genericErrorHandler maps errors (including pkg/errors.AppError) to HTTP responses
// using the full errors.HTTPStatus code map.
func genericErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	msg := "internal server error"
	payload := map[string]interface{}{}

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		switch m := he.Message.(type) {
		case string:
			msg = m
		default:
			msg = he.Error()
		}
		payload["error"] = msg
		payload["code"] = code
	} else {
		code = errors.HTTPStatus(err)
		var appErr *errors.AppError
		if errors.As(err, &appErr) {
			msg = appErr.Message
			payload["error"] = msg
			payload["code"] = appErr.Code
		} else {
			if err != nil {
				msg = err.Error()
			}
			payload["error"] = msg
			payload["code"] = code
		}
	}

	_ = c.JSON(code, payload)
}
