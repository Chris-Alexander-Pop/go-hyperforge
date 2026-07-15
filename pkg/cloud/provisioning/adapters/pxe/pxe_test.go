package pxe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/provisioning"
)

func TestNewRequiresBaseURL(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPXEProvisionDeprovisionPower(t *testing.T) {
	var mu sync.Mutex
	hosts := map[string]string{}
	var cycled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/hosts/h1/provision":
			hosts["h1"] = "busy"
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "busy"})
		case r.Method == http.MethodPost && r.URL.Path == "/hosts/h1/deprovision":
			hosts["h1"] = "offline"
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/hosts/h1/status":
			st := hosts["h1"]
			if st == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// After provision, simulate boot complete on status poll.
			if st == "busy" {
				st = "ready"
				hosts["h1"] = st
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": st})
		case r.Method == http.MethodPost && r.URL.Path == "/hosts/h1/power":
			cycled = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/hosts/missing/status":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p, err := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := p.ProvisionHost(ctx, "h1", "http://images/os.iso"); err != nil {
		t.Fatal(err)
	}
	st, err := p.GetHostStatus(ctx, "h1")
	if err != nil || st != cloud.HostStatusReady {
		t.Fatalf("status: %v %s", err, st)
	}
	if err := p.PowerCycle(ctx, "h1"); err != nil || !cycled {
		t.Fatalf("cycle: %v called=%v", err, cycled)
	}
	if err := p.DeprovisionHost(ctx, "h1"); err != nil {
		t.Fatal(err)
	}
	st, err = p.GetHostStatus(ctx, "h1")
	if err != nil || st != cloud.HostStatusOffline {
		t.Fatalf("after deprov: %v %s", err, st)
	}

	_, err = p.GetHostStatus(ctx, "missing")
	if err != provisioning.ErrHostNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestImplementsProvisioner(t *testing.T) {
	p, err := New(Config{BaseURL: "http://localhost"})
	if err != nil {
		t.Fatal(err)
	}
	var _ provisioning.Provisioner = p
}
