package etcd_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery/adapters/etcd"
	"github.com/stretchr/testify/require"
)

type fakeEtcd struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newFakeEtcd() *fakeEtcd {
	return &fakeEtcd{data: make(map[string][]byte)}
}

func (f *fakeEtcd) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/kv/put", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		k, _ := base64.StdEncoding.DecodeString(req.Key)
		v, _ := base64.StdEncoding.DecodeString(req.Value)
		f.mu.Lock()
		f.data[string(k)] = v
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})
	mux.HandleFunc("/v3/kv/range", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key      string `json:"key"`
			RangeEnd string `json:"range_end"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		start, _ := base64.StdEncoding.DecodeString(req.Key)
		end := []byte(nil)
		if req.RangeEnd != "" {
			end, _ = base64.StdEncoding.DecodeString(req.RangeEnd)
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		type kv struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		var kvs []kv
		for k, v := range f.data {
			kb := []byte(k)
			if end == nil {
				if k == string(start) {
					kvs = append(kvs, kv{
						Key:   base64.StdEncoding.EncodeToString(kb),
						Value: base64.StdEncoding.EncodeToString(v),
					})
				}
				continue
			}
			if string(kb) >= string(start) && string(kb) < string(end) {
				kvs = append(kvs, kv{
					Key:   base64.StdEncoding.EncodeToString(kb),
					Value: base64.StdEncoding.EncodeToString(v),
				})
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"kvs": kvs})
	})
	mux.HandleFunc("/v3/kv/deleterange", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key string `json:"key"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		k, _ := base64.StdEncoding.DecodeString(req.Key)
		f.mu.Lock()
		delete(f.data, string(k))
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})
	return mux
}

func TestEtcdRegistry_CRUD(t *testing.T) {
	fake := newFakeEtcd()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	reg, err := etcd.New(etcd.Config{Address: srv.URL, WatchInterval: 50 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })

	ctx := context.Background()
	svc, err := reg.Register(ctx, discovery.RegisterOptions{
		Name:    "api",
		Address: "10.0.0.1",
		Port:    8080,
		Tags:    []string{"v1"},
		Weight:  2,
	})
	require.NoError(t, err)
	require.Equal(t, "api", svc.Name)
	require.NotEmpty(t, svc.ID)

	got, err := reg.Get(ctx, svc.ID)
	require.NoError(t, err)
	require.Equal(t, "10.0.0.1", got.Address)

	list, err := reg.Lookup(ctx, "api", discovery.QueryOptions{})
	require.NoError(t, err)
	require.Len(t, list, 1)

	all, err := reg.List(ctx, discovery.QueryOptions{Tag: "v1"})
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, reg.UpdateHealth(ctx, svc.ID, discovery.HealthStatusWarning))
	require.NoError(t, reg.Heartbeat(ctx, svc.ID))

	require.NoError(t, reg.Deregister(ctx, svc.ID))
	_, err = reg.Get(ctx, svc.ID)
	require.ErrorIs(t, err, discovery.ErrServiceNotFound)
}

func TestEtcdRegistry_Watch(t *testing.T) {
	fake := newFakeEtcd()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	reg, err := etcd.New(etcd.Config{Address: srv.URL, WatchInterval: 30 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := reg.Watch(ctx, "web")
	require.NoError(t, err)

	go func() {
		time.Sleep(40 * time.Millisecond)
		_, _ = reg.Register(context.Background(), discovery.RegisterOptions{
			Name: "web", Address: "10.0.0.2", Port: 80,
		})
	}()

	var seen bool
	for snap := range ch {
		if len(snap) > 0 {
			seen = true
			cancel()
			break
		}
	}
	require.True(t, seen)
}
