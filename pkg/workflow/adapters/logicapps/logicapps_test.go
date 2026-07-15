package logicapps_test

import (
	"context"
	"encoding/json"
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
