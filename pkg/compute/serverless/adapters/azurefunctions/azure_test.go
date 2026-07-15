package azurefunctions

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/serverless"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestCRUDUnimplemented(t *testing.T) {
	rt, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ops := []error{
		func() error { _, e := rt.CreateFunction(ctx, serverless.CreateFunctionOptions{}); return e }(),
		func() error { _, e := rt.GetFunction(ctx, "x"); return e }(),
		func() error { _, e := rt.ListFunctions(ctx); return e }(),
		func() error { _, e := rt.UpdateFunction(ctx, "x", serverless.CreateFunctionOptions{}); return e }(),
		rt.DeleteFunction(ctx, "x"),
		func() error { _, e := rt.Invoke(ctx, serverless.InvokeOptions{FunctionName: "f"}); return e }(),
	}
	for i, err := range ops {
		if !pkgerrors.IsCode(err, pkgerrors.CodeUnimplemented) {
			t.Fatalf("op %d: expected Unimplemented, got %v", i, err)
		}
	}
}

func TestInvokeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/hello") {
			t.Fatalf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"echo":` + string(body) + `}`))
	}))
	defer srv.Close()

	rt, err := New(Config{InvokeBaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	res, err := rt.Invoke(context.Background(), serverless.InvokeOptions{
		FunctionName: "hello",
		Payload:      []byte(`"hi"`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || !strings.Contains(string(res.Payload), "hi") {
		t.Fatalf("unexpected: %+v", res)
	}
}
