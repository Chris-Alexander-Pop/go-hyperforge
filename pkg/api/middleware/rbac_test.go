package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/middleware"
	"github.com/chris-alexander-pop/system-design-library/pkg/api/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withRoles(r *http.Request, roles []string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextKeyRoles, roles)
	ctx = context.WithValue(ctx, middleware.ContextKeySubject, "user-1")
	return r.WithContext(ctx)
}

func TestRequirePermission_AllowsMatchingRole(t *testing.T) {
	e := rbac.New()
	e.AddPolicy("editor", "posts", "write")

	h := middleware.RequirePermission(e, "posts", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := withRoles(httptest.NewRequest(http.MethodPost, "/posts", nil), []string{"editor"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermission_DeniesMissingPermission(t *testing.T) {
	e := rbac.New()
	e.AddPolicy("viewer", "posts", "read")

	h := middleware.RequirePermission(e, "posts", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := withRoles(httptest.NewRequest(http.MethodPost, "/posts", nil), []string{"viewer"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermission_DeniesNoRoles(t *testing.T) {
	e := rbac.New()
	h := middleware.RequirePermission(e, "posts", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermission_AdminBypass(t *testing.T) {
	e := rbac.New()
	h := middleware.RequirePermission(e, "secrets", "delete")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := withRoles(httptest.NewRequest(http.MethodDelete, "/secrets", nil), []string{"admin"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRateLimitKeyByUserAndAPIKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	assert.Equal(t, "ip:10.0.0.1", middleware.KeyByUser(req))

	req = withRoles(req, []string{"r"})
	assert.Equal(t, "user:user-1", middleware.KeyByUser(req))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.2:1"
	req2.Header.Set("X-API-Key", "secret")
	assert.Equal(t, "apikey:secret", middleware.KeyByAPIKey(req2))
}
