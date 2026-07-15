package openapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/openapi"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentStub(t *testing.T) {
	doc := openapi.NewDocument("Demo", "1.0.0")
	if err := doc.AddOperation("/health", "get", openapi.Operation{
		OperationID: "healthCheck",
		Summary:     "Liveness",
	}); err != nil {
		t.Fatalf("AddOperation: %v", err)
	}
	raw, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["openapi"] != "3.0.3" {
		t.Fatalf("openapi=%v", got["openapi"])
	}
	paths, ok := got["paths"].(map[string]interface{})
	if !ok || paths["/health"] == nil {
		t.Fatalf("paths=%v", got["paths"])
	}
}

func TestFromRoutes(t *testing.T) {
	doc, err := openapi.FromRoutes("Users", "2.0.0", []openapi.RouteMeta{
		{
			Path:        "/users/{id}",
			Method:      "GET",
			OperationID: "getUser",
			Summary:     "Get user",
			Tags:        []string{"users"},
		},
		{
			Path:        "/users",
			Method:      "POST",
			OperationID: "createUser",
			Summary:     "Create user",
			Tags:        []string{"users"},
			Responses: map[string]*openapi.Response{
				"201": {Description: "Created"},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc.Paths["/users/{id}"].Get)
	require.NotNil(t, doc.Paths["/users"].Post)
	assert.Equal(t, "getUser", doc.Paths["/users/{id}"].Get.OperationID)
	assert.Len(t, doc.Paths["/users/{id}"].Get.Parameters, 1)
	assert.Equal(t, "id", doc.Paths["/users/{id}"].Get.Parameters[0].Name)
	assert.Equal(t, "path", doc.Paths["/users/{id}"].Get.Parameters[0].In)
	assert.Equal(t, "201", mustKey(doc.Paths["/users"].Post.Responses, "201"))
	assert.Len(t, doc.Tags, 1)
	assert.Equal(t, "users", doc.Tags[0].Name)

	raw, err := doc.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"openapi":"3.0.3"`)
}

func mustKey(m map[string]*openapi.Response, k string) string {
	if _, ok := m[k]; ok {
		return k
	}
	return ""
}

func TestEchoStdBridge(t *testing.T) {
	e := echo.New()

	var sawAuth bool
	authMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sawAuth = true
			next.ServeHTTP(w, r)
		})
	}

	std := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	openapi.MountStd(e, http.MethodGet, "/bridged", openapi.ChainStd(std, authMW))

	req := httptest.NewRequest(http.MethodGet, "/bridged", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
	assert.True(t, sawAuth)

	// StdHandler: Echo handler → net/http
	eh := func(c echo.Context) error {
		return c.String(http.StatusCreated, "echo")
	}
	h := openapi.StdHandler(e, eh)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusCreated, rec2.Code)
	assert.Equal(t, "echo", rec2.Body.String())

	// EchoHandler wraps stdlib
	wrapped := openapi.EchoHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec3 := httptest.NewRecorder()
	c := e.NewContext(req3, rec3)
	require.NoError(t, wrapped(c))
	assert.Equal(t, http.StatusAccepted, rec3.Code)

	emw := openapi.EchoMiddleware(authMW)
	assert.NotNil(t, emw)
}
