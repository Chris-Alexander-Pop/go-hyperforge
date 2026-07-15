package consul_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery/adapters/consul"
	"github.com/stretchr/testify/require"
)

type fakeConsul struct {
	mu       sync.Mutex
	services map[string]map[string]any
	checks   map[string]string
	token    string
}

func newFakeConsul() *fakeConsul {
	return &fakeConsul{
		services: make(map[string]map[string]any),
		checks:   make(map[string]string),
	}
}

func (f *fakeConsul) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.token != "" && r.Header.Get("X-Consul-Token") != f.token {
			http.Error(w, "ACL not found", http.StatusForbidden)
			return
		}
		path := r.URL.Path
		switch {
		case path == "/v1/agent/service/register":
			f.register(w, r)
		case strings.HasPrefix(path, "/v1/agent/service/deregister/"):
			f.deregister(w, r)
		case path == "/v1/agent/services":
			f.listServices(w, r)
		case strings.HasPrefix(path, "/v1/agent/service/"):
			f.getService(w, r)
		case strings.HasPrefix(path, "/v1/health/service/"):
			f.healthService(w, r)
		case strings.HasPrefix(path, "/v1/agent/check/pass/service:"):
			f.checkPass(w, r)
		case strings.HasPrefix(path, "/v1/agent/check/warn/service:"):
			f.checkWarn(w, r)
		case strings.HasPrefix(path, "/v1/agent/check/fail/service:"):
			f.checkFail(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func (f *fakeConsul) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, _ := body["ID"].(string)
	if id == "" {
		http.Error(w, "missing ID", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.services[id] = body
	f.checks[id] = "passing"
	f.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (f *fakeConsul) deregister(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/agent/service/deregister/")
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.services[id]; !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	delete(f.services, id)
	delete(f.checks, id)
	w.WriteHeader(http.StatusOK)
}

func (f *fakeConsul) getService(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/agent/service/")
	if strings.HasPrefix(id, "deregister/") {
		http.NotFound(w, r)
		return
	}
	f.mu.Lock()
	svc, ok := f.services[id]
	f.mu.Unlock()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(agentView(svc))
}

func (f *fakeConsul) listServices(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.services))
	for id, svc := range f.services {
		out[id] = agentView(svc)
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (f *fakeConsul) healthService(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/v1/health/service/")
	passingOnly := r.URL.Query().Get("passing") == "true"
	tag := r.URL.Query().Get("tag")

	f.mu.Lock()
	defer f.mu.Unlock()
	var entries []map[string]any
	for id, svc := range f.services {
		svcName, _ := svc["Name"].(string)
		if svcName != name {
			continue
		}
		status := f.checks[id]
		if passingOnly && status != "passing" {
			continue
		}
		if tag != "" {
			ok := false
			if tags, _ := svc["Tags"].([]any); tags != nil {
				for _, t := range tags {
					if s, _ := t.(string); s == tag {
						ok = true
						break
					}
				}
			}
			if !ok {
				continue
			}
		}
		entries = append(entries, map[string]any{
			"Service": agentView(svc),
			"Checks":  []map[string]string{{"Status": status}},
		})
	}
	w.Header().Set("X-Consul-Index", "1")
	_ = json.NewEncoder(w).Encode(entries)
}

func (f *fakeConsul) checkPass(w http.ResponseWriter, r *http.Request) {
	f.setCheck(w, r, "/v1/agent/check/pass/service:", "passing")
}
func (f *fakeConsul) checkWarn(w http.ResponseWriter, r *http.Request) {
	f.setCheck(w, r, "/v1/agent/check/warn/service:", "warning")
}
func (f *fakeConsul) checkFail(w http.ResponseWriter, r *http.Request) {
	f.setCheck(w, r, "/v1/agent/check/fail/service:", "critical")
}

func (f *fakeConsul) setCheck(w http.ResponseWriter, r *http.Request, prefix, status string) {
	id := strings.TrimPrefix(r.URL.Path, prefix)
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.services[id]; !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	f.checks[id] = status
	w.WriteHeader(http.StatusOK)
}

func agentView(svc map[string]any) map[string]any {
	id, _ := svc["ID"].(string)
	name, _ := svc["Name"].(string)
	addr, _ := svc["Address"].(string)
	port, _ := svc["Port"].(float64)
	tags, _ := svc["Tags"].([]any)
	meta, _ := svc["Meta"].(map[string]any)
	weights, _ := svc["Weights"].(map[string]any)
	tagStrs := make([]string, 0, len(tags))
	for _, t := range tags {
		if s, ok := t.(string); ok {
			tagStrs = append(tagStrs, s)
		}
	}
	metaStr := map[string]string{}
	for k, v := range meta {
		if s, ok := v.(string); ok {
			metaStr[k] = s
		}
	}
	out := map[string]any{
		"ID":      id,
		"Service": name,
		"Address": addr,
		"Port":    int(port),
		"Tags":    tagStrs,
		"Meta":    metaStr,
	}
	if weights != nil {
		out["Weights"] = weights
	}
	return out
}

func TestConsulRegisterLookupGetDeregister(t *testing.T) {
	fake := newFakeConsul()
	fake.token = "secret"
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	reg, err := consul.New(consul.Config{
		Address:    srv.URL,
		Token:      "secret",
		HTTPClient: srv.Client(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })

	ctx := context.Background()
	svc, err := reg.Register(ctx, discovery.RegisterOptions{
		ID:       "api-1",
		Name:     "api",
		Address:  "10.0.0.1",
		Port:     8080,
		Tags:     []string{"v1"},
		Metadata: map[string]string{"env": "test"},
		Weight:   5,
		TTL:      30 * time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, "api-1", svc.ID)

	got, err := reg.Get(ctx, "api-1")
	require.NoError(t, err)
	require.Equal(t, "api", got.Name)
	require.Equal(t, 8080, got.Port)
	require.Equal(t, 5, got.Weight)

	list, err := reg.Lookup(ctx, "api", discovery.QueryOptions{HealthyOnly: true, Tag: "v1"})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "api-1", list[0].ID)

	all, err := reg.List(ctx, discovery.QueryOptions{})
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, reg.Heartbeat(ctx, "api-1"))
	require.NoError(t, reg.UpdateHealth(ctx, "api-1", discovery.HealthStatusWarning))

	require.NoError(t, reg.Deregister(ctx, "api-1"))
	_, err = reg.Get(ctx, "api-1")
	require.ErrorIs(t, err, discovery.ErrServiceNotFound)
}

func TestConsulMissingName(t *testing.T) {
	srv := httptest.NewServer(newFakeConsul().handler())
	t.Cleanup(srv.Close)
	reg, err := consul.New(consul.Config{Address: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)
	_, err = reg.Register(context.Background(), discovery.RegisterOptions{Address: "10.0.0.1"})
	require.ErrorIs(t, err, discovery.ErrInvalidService)
}

func TestConsulACLRejected(t *testing.T) {
	fake := newFakeConsul()
	fake.token = "secret"
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	reg, err := consul.New(consul.Config{Address: srv.URL, Token: "wrong", HTTPClient: srv.Client()})
	require.NoError(t, err)
	_, err = reg.Register(context.Background(), discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.1", Port: 80,
	})
	require.Error(t, err)
}

func TestConsulWatch(t *testing.T) {
	fake := newFakeConsul()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	reg, err := consul.New(consul.Config{
		Address:    srv.URL,
		HTTPClient: srv.Client(),
		WatchWait:  50 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = reg.Register(ctx, discovery.RegisterOptions{
		ID: "w1", Name: "watched", Address: "10.0.0.2", Port: 9,
	})
	require.NoError(t, err)

	ch, err := reg.Watch(ctx, "watched")
	require.NoError(t, err)

	select {
	case services := <-ch:
		require.NotEmpty(t, services)
		require.Equal(t, "w1", services[0].ID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch")
	}
	cancel()
	for range ch {
	}
}

func TestConsulEmptyAddress(t *testing.T) {
	_, err := consul.New(consul.Config{})
	require.Error(t, err)
}

func TestConsulHeartbeatNotFound(t *testing.T) {
	srv := httptest.NewServer(newFakeConsul().handler())
	t.Cleanup(srv.Close)
	reg, err := consul.New(consul.Config{Address: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)
	err = reg.Heartbeat(context.Background(), "missing")
	require.ErrorIs(t, err, discovery.ErrServiceNotFound)
}
