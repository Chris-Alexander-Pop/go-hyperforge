package openapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// StdMiddleware is a standard library middleware function.
type StdMiddleware func(http.Handler) http.Handler

// EchoMiddleware converts a net/http middleware into Echo middleware via echo.WrapMiddleware.
func EchoMiddleware(mw StdMiddleware) echo.MiddlewareFunc {
	return echo.WrapMiddleware(mw)
}

// StdHandler converts an Echo handler into a net/http HandlerFunc.
// The Echo instance is used only to create a context for the handler.
func StdHandler(e *echo.Echo, h echo.HandlerFunc) http.HandlerFunc {
	if e == nil {
		e = echo.New()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		c := e.NewContext(r, w)
		if err := h(c); err != nil {
			e.HTTPErrorHandler(err, c)
		}
	}
}

// EchoHandler wraps a net/http Handler as an Echo handler via echo.WrapHandler.
func EchoHandler(h http.Handler) echo.HandlerFunc {
	return echo.WrapHandler(h)
}

// EchoHandlerFunc wraps a net/http HandlerFunc as an Echo handler.
func EchoHandlerFunc(h http.HandlerFunc) echo.HandlerFunc {
	return echo.WrapHandler(h)
}

// ChainStd applies stdlib middlewares around a handler (outer-first).
func ChainStd(h http.Handler, mws ...StdMiddleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// MountStd mounts a net/http Handler on an Echo route group/path using WrapHandler.
func MountStd(e *echo.Echo, method, path string, h http.Handler, mws ...echo.MiddlewareFunc) *echo.Route {
	return e.Add(method, path, echo.WrapHandler(h), mws...)
}
