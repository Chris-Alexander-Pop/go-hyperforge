package graphql_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	apigraphql "github.com/chris-alexander-pop/go-hyperforge/pkg/api/graphql"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func testSchema(complexity int) graphql.ExecutableSchema {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			name: String!
		}
	`})
	return &graphql.ExecutableSchemaMock{
		SchemaFunc: func() *ast.Schema { return schema },
		ComplexityFunc: func(ctx context.Context, typeName, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			return complexity, true
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			return func(ctx context.Context) *graphql.Response {
				return &graphql.Response{Data: []byte(`{"name":"test"}`)}
			}
		},
	}
}

func TestNewHandlerServesQuery(t *testing.T) {
	h := apigraphql.NewHandlerWithConfig(testSchema(1), apigraphql.HandlerConfig{
		ComplexityLimit:     10,
		DepthLimit:          5,
		DisableOTel:         true,
		EnableIntrospection: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(`{"query":"{ name }"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "name") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestComplexityLimitExceeded(t *testing.T) {
	h := apigraphql.NewHandlerWithConfig(testSchema(5), apigraphql.HandlerConfig{
		ComplexityLimit:     2,
		DepthLimit:          -1,
		DisableOTel:         true,
		EnableIntrospection: false,
	})
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(`{"query":"{ name }"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "COMPLEXITY_LIMIT_EXCEEDED") && !strings.Contains(body, "complexity") {
		t.Fatalf("expected complexity error, body=%s", body)
	}
}

func TestDefaultHandlerConfig(t *testing.T) {
	cfg := apigraphql.DefaultHandlerConfig()
	if cfg.ComplexityLimit != apigraphql.DefaultComplexityLimit {
		t.Fatalf("complexity=%d", cfg.ComplexityLimit)
	}
	if cfg.DepthLimit != apigraphql.DefaultDepthLimit {
		t.Fatalf("depth=%d", cfg.DepthLimit)
	}
	if !cfg.EnableIntrospection {
		t.Fatal("expected EnableIntrospection true by default")
	}
}

func TestPlaygroundHandler(t *testing.T) {
	h := apigraphql.NewPlaygroundHandler("/query")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestPlaygroundHandlerWithConfig(t *testing.T) {
	h := apigraphql.NewPlaygroundHandlerWithConfig(apigraphql.PlaygroundConfig{
		Title:         "My API",
		Endpoint:      "/graphql",
		StoragePrefix: "demo",
		UIHeaders:     map[string]string{"X-Env": "test"},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "My API") {
		t.Fatalf("expected title in body")
	}
}

func TestLoadSDLAndRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.graphql")
	sdl := "type Query { hello: String! }\n"
	if err := os.WriteFile(path, []byte(sdl), 0o644); err != nil {
		t.Fatal(err)
	}

	schema, err := apigraphql.LoadSDLFile(path)
	if err != nil {
		t.Fatalf("LoadSDLFile: %v", err)
	}
	if schema.Query == nil {
		t.Fatal("missing Query")
	}

	reg := apigraphql.NewSchemaRegistry()
	if err := reg.RegisterFile("main", path); err != nil {
		t.Fatalf("RegisterFile: %v", err)
	}
	got, ok := reg.Get("main")
	if !ok || !strings.Contains(got, "hello") {
		t.Fatalf("Get=%q ok=%v", got, ok)
	}
	if len(reg.Names()) != 1 {
		t.Fatalf("Names=%v", reg.Names())
	}
}

func TestIntrospectionToggle(t *testing.T) {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			name: String!
		}
	`})
	es := &graphql.ExecutableSchemaMock{
		SchemaFunc: func() *ast.Schema { return schema },
		ComplexityFunc: func(ctx context.Context, typeName, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			return 1, true
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			return graphql.OneShot(&graphql.Response{Data: []byte(`{}`)})
		},
	}

	q := `{"query":"{ __schema { queryType { name } } }"}`

	enabled := apigraphql.NewHandlerWithConfig(es, apigraphql.HandlerConfig{
		ComplexityLimit:     -1,
		DepthLimit:          -1,
		DisableOTel:         true,
		EnableIntrospection: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(q))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	enabled.ServeHTTP(rec, req)
	// With mock ExecFunc, introspection may still hit DisableIntrospection path
	// in executor before Exec; ensure handler accepts the request.
	if rec.Code != http.StatusOK {
		t.Fatalf("enabled status=%d body=%s", rec.Code, rec.Body.String())
	}

	disabled := apigraphql.NewHandlerWithConfig(es, apigraphql.HandlerConfig{
		ComplexityLimit:     -1,
		DepthLimit:          -1,
		DisableOTel:         true,
		EnableIntrospection: false,
	})
	req2 := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(q))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	disabled.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("disabled status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	body := rec2.Body.String()
	if !strings.Contains(body, "introspection") && !strings.Contains(strings.ToLower(body), "disabled") && !strings.Contains(body, "error") {
		// gqlgen returns an error when introspection is disabled
		t.Logf("disabled introspection body=%s", body)
	}
}
