package libvirt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/hypervisor"
	"github.com/google/uuid"
)

func TestRemoteLibvirtLifecycle(t *testing.T) {
	store := map[string]hypervisor.VM{}
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/vms":
			var req createReq
			_ = json.NewDecoder(r.Body).Decode(&req)
			id := uuid.NewString()
			store[id] = hypervisor.VM{
				ID: id, Name: req.Spec.Name, Status: cloud.InstanceStatusRunning,
				Spec: req.Spec, CreatedAt: time.Now(),
			}
			_ = json.NewEncoder(w).Encode(createResp{ID: id})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/vms":
			list := make([]vmDTO, 0, len(store))
			for _, vm := range store {
				list = append(list, vmDTO{ID: vm.ID, Name: vm.Name, Status: vm.Status, Spec: vm.Spec, CreatedAt: vm.CreatedAt})
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/v1/vms/"):
			id := r.URL.Path[len("/v1/vms/"):]
			vm, ok := store[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_ = json.NewEncoder(w).Encode(vmDTO{ID: vm.ID, Name: vm.Name, Status: vm.Status, Spec: vm.Spec, CreatedAt: vm.CreatedAt})
		case r.Method == http.MethodPost && stringsHasSuffix(r.URL.Path, "/start"):
			id := r.URL.Path[len("/v1/vms/") : len(r.URL.Path)-len("/start")]
			vm, ok := store[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			vm.Status = cloud.InstanceStatusRunning
			store[id] = vm
			w.WriteHeader(204)
		case r.Method == http.MethodPost && stringsHasSuffix(r.URL.Path, "/stop"):
			id := r.URL.Path[len("/v1/vms/") : len(r.URL.Path)-len("/stop")]
			vm, ok := store[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			vm.Status = cloud.InstanceStatusStopped
			store[id] = vm
			w.WriteHeader(204)
		case r.Method == http.MethodDelete:
			id := r.URL.Path[len("/v1/vms/"):]
			if _, ok := store[id]; !ok {
				w.WriteHeader(404)
				return
			}
			delete(store, id)
			w.WriteHeader(204)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	h, err := New(Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id, err := h.CreateVM(ctx, hypervisor.VMSpec{Name: "vm1", MemoryMB: 1024, VCPUs: 2})
	if err != nil {
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
	if err := h.StartVM(ctx, id); err != nil {
		t.Fatal(err)
	}
	list, err := h.ListVMs(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if err := h.DeleteVM(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := h.GetVMStatus(ctx, id); err != hypervisor.ErrVMNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func TestNewRequiresBaseURL(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
}
