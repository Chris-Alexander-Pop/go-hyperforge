package firecracker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/hypervisor"
)

func TestFirecrackerCreateStartStop(t *testing.T) {
	calls := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	h, err := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id, err := h.CreateVM(ctx, hypervisor.VMSpec{Name: "fc1", MemoryMB: 256, VCPUs: 1, Image: "/vmlinux"})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.StartVM(ctx, id); err != nil {
		t.Fatal(err)
	}
	st, err := h.GetVMStatus(ctx, id)
	if err != nil || st != cloud.InstanceStatusRunning {
		t.Fatalf("status: %v %s", err, st)
	}
	if err := h.StopVM(ctx, id); err != nil {
		t.Fatal(err)
	}
	st, _ = h.GetVMStatus(ctx, id)
	if st != cloud.InstanceStatusStopped {
		t.Fatalf("expected stopped, got %s", st)
	}
	list, err := h.ListVMs(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if err := h.DeleteVM(ctx, id); err != nil {
		t.Fatal(err)
	}

	foundMachine := false
	foundStart := false
	for _, c := range calls {
		if c == "PUT /machine-config" {
			foundMachine = true
		}
		if c == "PUT /actions" {
			foundStart = true
		}
	}
	if !foundMachine || !foundStart {
		t.Fatalf("expected machine-config and actions calls, got %v", calls)
	}
}

func TestFirecrackerDuplicateName(t *testing.T) {
	h, _ := New(Config{})
	ctx := context.Background()
	if _, err := h.CreateVM(ctx, hypervisor.VMSpec{Name: "a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.CreateVM(ctx, hypervisor.VMSpec{Name: "a"}); err != hypervisor.ErrVMAlreadyExists {
		t.Fatalf("expected ErrVMAlreadyExists, got %v", err)
	}
}
