package graphql_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
		ComplexityLimit: 10,
		DepthLimit:      5,
		DisableOTel:     true,
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
		ComplexityLimit: 2,
		DepthLimit:      -1,
		DisableOTel:     true,
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
