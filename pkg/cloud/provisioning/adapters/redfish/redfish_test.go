package redfish

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestRedfishPowerCycleAndStatus(t *testing.T) {
	var resetCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/Systems/node-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"PowerState": "On",
				"Status":     map[string]string{"State": "Enabled"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/Systems/node-1/Actions/ComputerSystem.Reset":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["ResetType"] != "ForceRestart" {
				t.Fatalf("unexpected reset: %v", body)
			}
			resetCalled = true
			w.WriteHeader(204)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	p, err := New(Config{
		BaseURL:     srv.URL,
		HTTPClient:  srv.Client(),
		HostSystems: map[string]string{"host-a": "/Systems/node-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, err := p.GetHostStatus(ctx, "host-a")
	if err != nil || st != cloud.HostStatusReady {
		t.Fatalf("status: %v %s", err, st)
	}
	if err := p.PowerCycle(ctx, "host-a"); err != nil {
		t.Fatal(err)
	}
	if !resetCalled {
		t.Fatal("expected reset action")
	}
	if err := p.ProvisionHost(ctx, "host-a", "http://img"); !pkgerrors.IsCode(err, pkgerrors.CodeUnimplemented) {
		t.Fatalf("expected Unimplemented, got %v", err)
	}
}

func TestRedfishNewRequiresURL(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
}
