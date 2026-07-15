package ipmi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
)

func TestIPMIPowerCycle(t *testing.T) {
	var cycled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/hosts/h1/power":
			_ = json.NewEncoder(w).Encode(map[string]string{"power": "on"})
		case r.Method == http.MethodPost && r.URL.Path == "/hosts/h1/power":
			cycled = true
			w.WriteHeader(204)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	p, err := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	st, err := p.GetHostStatus(context.Background(), "h1")
	if err != nil || st != cloud.HostStatusReady {
		t.Fatalf("status: %v %s", err, st)
	}
	if err := p.PowerCycle(context.Background(), "h1"); err != nil || !cycled {
		t.Fatalf("cycle: %v called=%v", err, cycled)
	}
}
