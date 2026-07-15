package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/rest"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPStatusErrorMapping(t *testing.T) {
	srv := rest.New(rest.Config{Port: "0"})
	e := srv.Echo()

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not_found", errors.NotFound("missing", nil), http.StatusNotFound, errors.CodeNotFound},
		{"invalid", errors.InvalidArgument("bad", nil), http.StatusBadRequest, errors.CodeInvalidArgument},
		{"unauthorized", errors.Unauthorized("nope", nil), http.StatusUnauthorized, errors.CodeUnauthorized},
		{"forbidden", errors.Forbidden("deny", nil), http.StatusForbidden, errors.CodeForbidden},
		{"conflict", errors.Conflict("exists", nil), http.StatusConflict, errors.CodeConflict},
		{"unimplemented", errors.Unimplemented("todo", nil), http.StatusNotImplemented, errors.CodeUnimplemented},
		{"deadline", errors.DeadlineExceeded("slow", nil), http.StatusGatewayTimeout, errors.CodeDeadlineExceeded},
		{"unavailable", errors.Unavailable("down", nil), http.StatusServiceUnavailable, errors.CodeUnavailable},
		{"exhausted", errors.ResourceExhausted("limit", nil), http.StatusTooManyRequests, errors.CodeResourceExhausted},
		{"canceled", errors.Canceled("bye", nil), errors.StatusClientClosedRequest, errors.CodeCanceled},
		{"wrapped", errors.Wrap(errors.NotFound("x", nil), "outer"), http.StatusNotFound, errors.CodeNotFound},
		{"internal", errors.Internal("boom", nil), http.StatusInternalServerError, errors.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/" + tc.name
			e.GET(path, func(c echo.Context) error {
				return tc.err
			})

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tc.wantCode, body["code"])
		})
	}
}

func TestReadWriteTimeoutsApplied(t *testing.T) {
	srv := rest.New(rest.Config{
		Port:         "0",
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 7 * time.Second,
	})
	assert.Equal(t, 3*time.Second, srv.Echo().Server.ReadTimeout)
	assert.Equal(t, 7*time.Second, srv.Echo().Server.WriteTimeout)
}

func TestEchoHTTPErrorStillMapped(t *testing.T) {
	srv := rest.New(rest.Config{Port: "0"})
	e := srv.Echo()
	e.GET("/he", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusTeapot, "short and stout")
	})

	req := httptest.NewRequest(http.MethodGet, "/he", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}
