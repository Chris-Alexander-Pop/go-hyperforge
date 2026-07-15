package logicapps_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/logicapps"
)

func TestRemoteRunStatusAndClose(t *testing.T) {
	var gotRunPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/triggers/manual/run") && r.Method == http.MethodPost:
			w.Header().Set("x-ms-workflow-run-id", "run-42")
			w.WriteHeader(http.StatusAccepted)
		case strings.Contains(r.URL.Path, "/runs/run-42") && r.Method == http.MethodGet:
			gotRunPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "run-42",
				"properties": map[string]interface{}{
					"startTime": time.Unix(10, 0).UTC().Format(time.RFC3339),
					"endTime":   time.Unix(20, 0).UTC().Format(time.RFC3339),
					"status":    "Succeeded",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		SkipAuth:       true,
		HTTPClient:     srv.Client(),
		ManagementBase: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	exec, err := eng.Start(context.Background(), workflow.StartOptions{WorkflowID: "app1", Input: map[string]string{"a": "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if exec.ID != "run-42" {
		t.Fatalf("id=%q", exec.ID)
	}

	got, err := eng.GetExecution(context.Background(), "run-42")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != workflow.StatusCompleted {
		t.Fatalf("status=%s path=%s", got.Status, gotRunPath)
	}
	if !strings.Contains(gotRunPath, "/workflows/app1/runs/run-42") {
		t.Fatalf("unexpected path %s", gotRunPath)
	}

	if err := eng.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = eng.GetExecution(context.Background(), "run-42")
	if err == nil {
		t.Fatal("expected closed error")
	}
}

func TestClientSecretAuth(t *testing.T) {
	var gotForm string
	var gotAuth string
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/oauth2/v2.0/token"):
			body, _ := io.ReadAll(r.Body)
			gotForm = string(body)
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "aad-token-xyz"})
		case strings.Contains(r.URL.Path, "/triggers/manual/run"):
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("x-ms-workflow-run-id", "run-cs")
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	})

	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		TenantID:       "tenant",
		ClientID:       "app-id",
		ClientSecret:   "secret",
		AuthMode:       logicapps.AuthModeClientSecret,
		LoginBase:      srv.URL,
		ManagementBase: srv.URL,
		HTTPClient:     srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotForm, "grant_type=client_credentials") {
		t.Fatalf("form=%q", gotForm)
	}
	if !strings.Contains(gotForm, "client_id=app-id") {
		t.Fatalf("form missing client_id: %q", gotForm)
	}
	if !strings.Contains(gotForm, "scope=") {
		t.Fatalf("form missing scope: %q", gotForm)
	}
	if _, err := eng.Start(context.Background(), workflow.StartOptions{WorkflowID: "app1"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer aad-token-xyz" {
		t.Fatalf("Authorization=%q", gotAuth)
	}
}

func TestManagedIdentityIMDSAuth(t *testing.T) {
	var gotIMDSPath string
	var gotIMDSQuery string
	var gotMetadata string
	var gotAuth string

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/metadata/identity/oauth2/token"):
			gotIMDSPath = r.URL.Path
			gotIMDSQuery = r.URL.RawQuery
			gotMetadata = r.Header.Get("Metadata")
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "msi-token-99"})
		case strings.Contains(r.URL.Path, "/triggers/manual/run"):
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("x-ms-workflow-run-id", "run-msi")
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	})

	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID:          "sub",
		ResourceGroup:           "rg",
		AuthMode:                logicapps.AuthModeManagedIdentity,
		ManagedIdentityClientID: "uai-client-id",
		IdentityBase:            srv.URL,
		ManagementBase:          srv.URL,
		HTTPClient:              srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMetadata != "true" {
		t.Fatalf("Metadata header=%q", gotMetadata)
	}
	if !strings.Contains(gotIMDSPath, "/metadata/identity/oauth2/token") {
		t.Fatalf("path=%q", gotIMDSPath)
	}
	if !strings.Contains(gotIMDSQuery, "api-version=") {
		t.Fatalf("query=%q", gotIMDSQuery)
	}
	if !strings.Contains(gotIMDSQuery, "resource=") {
		t.Fatalf("query missing resource: %q", gotIMDSQuery)
	}
	if !strings.Contains(gotIMDSQuery, "client_id=uai-client-id") {
		t.Fatalf("query missing client_id: %q", gotIMDSQuery)
	}
	if strings.Contains(gotIMDSQuery, "/.default") {
		t.Fatalf("IMDS resource should not include /.default: %q", gotIMDSQuery)
	}

	if _, err := eng.Start(context.Background(), workflow.StartOptions{WorkflowID: "app1"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer msi-token-99" {
		t.Fatalf("Authorization=%q", gotAuth)
	}
}

func TestUseManagedIdentityFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/metadata/identity/oauth2/token") {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "msi-via-flag"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID:     "sub",
		ResourceGroup:      "rg",
		UseManagedIdentity: true,
		IdentityBase:       srv.URL,
		HTTPClient:         srv.Client(),
		ManagementBase:     srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = eng
}

func TestTokenSourceOverride(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("x-ms-workflow-run-id", "run-ts")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		AuthMode:       logicapps.AuthModeClientSecret, // overridden by TokenSource
		TokenSource: logicapps.TokenSourceFunc(func(ctx context.Context) (string, error) {
			return "injected-token", nil
		}),
		ManagementBase: srv.URL,
		HTTPClient:     srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Start(context.Background(), workflow.StartOptions{WorkflowID: "app1"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer injected-token" {
		t.Fatalf("Authorization=%q", gotAuth)
	}
}

func TestSkipAuthNoTokenFetch(t *testing.T) {
	var authHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/oauth2/") || strings.Contains(r.URL.Path, "/metadata/identity/") {
			authHits++
			http.Error(w, "should not auth", http.StatusInternalServerError)
			return
		}
		w.Header().Set("x-ms-workflow-run-id", "run-skip")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		SkipAuth:       true,
		LoginBase:      srv.URL,
		IdentityBase:   srv.URL,
		ManagementBase: srv.URL,
		HTTPClient:     srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if authHits != 0 {
		t.Fatalf("authHits=%d", authHits)
	}
	if _, err := eng.Start(context.Background(), workflow.StartOptions{WorkflowID: "app1"}); err != nil {
		t.Fatal(err)
	}
}

func TestClosedEngineUnchanged(t *testing.T) {
	eng, err := logicapps.New(logicapps.Config{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		SkipAuth:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.Close(); err != nil {
		t.Fatal(err)
	}
	if err := eng.Close(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := eng.RegisterWorkflow(ctx, workflow.WorkflowDefinition{ID: "x"}); err == nil {
		t.Fatal("expected closed on RegisterWorkflow")
	}
	if _, err := eng.GetWorkflow(ctx, "x"); err == nil {
		t.Fatal("expected closed on GetWorkflow")
	}
	if _, err := eng.Start(ctx, workflow.StartOptions{WorkflowID: "x"}); err == nil {
		t.Fatal("expected closed on Start")
	}
	if _, err := eng.GetExecution(ctx, "x"); err == nil {
		t.Fatal("expected closed on GetExecution")
	}
	if _, err := eng.ListExecutions(ctx, workflow.ListOptions{}); err == nil {
		t.Fatal("expected closed on ListExecutions")
	}
	if err := eng.Cancel(ctx, "x"); err == nil {
		t.Fatal("expected closed on Cancel")
	}
}

func TestUnknownAuthMode(t *testing.T) {
	_, err := logicapps.New(logicapps.Config{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		AuthMode:       "nope",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
