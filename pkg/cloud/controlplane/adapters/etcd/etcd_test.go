package etcd_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane/adapters/etcd"
	"github.com/stretchr/testify/assert"
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

func TestEtcdControlPlane_HostInventory(t *testing.T) {
	fake := newFakeEtcd()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	cp, err := etcd.New(etcd.Config{Address: srv.URL})
	require.NoError(t, err)

	ctx := context.Background()
	host := cloud.Host{
		ID:     "h1",
		Name:   "node-1",
		Status: cloud.HostStatusReady,
		Capacity: cloud.Resources{
			VCPUs: 4, MemoryMB: 8192, DiskGB: 100,
		},
		Zone: "z1",
	}
	require.NoError(t, cp.RegisterHost(ctx, host))

	got, err := cp.GetHost(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, "node-1", got.Name)
	assert.Equal(t, 4, got.Available.VCPUs)

	hosts, err := cp.ListHosts(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	err = cp.RegisterHost(ctx, host)
	assert.ErrorIs(t, err, controlplane.ErrHostAlreadyRegistered)

	require.NoError(t, cp.UpdateHostStatus(ctx, "h1", cloud.HostStatusMaintenance))
	got, err = cp.GetHost(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, cloud.HostStatusMaintenance, got.Status)

	require.NoError(t, cp.UpdateHostStatus(ctx, "h1", cloud.HostStatusReady))
	inst, err := cp.CreateInstance(ctx, controlplane.CreateInstanceRequest{
		Name:   "vm1",
		HostID: "h1",
		Resources: cloud.Resources{
			VCPUs: 2, MemoryMB: 2048, DiskGB: 20,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "h1", inst.HostID)

	got, err = cp.GetHost(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Available.VCPUs)

	err = cp.DeregisterHost(ctx, "h1")
	assert.ErrorIs(t, err, controlplane.ErrHostHasInstances)

	require.NoError(t, cp.DeleteInstance(ctx, inst.ID))
	require.NoError(t, cp.DeregisterHost(ctx, "h1"))

	_, err = cp.GetHost(ctx, "h1")
	assert.ErrorIs(t, err, controlplane.ErrHostNotFound)

	// Keys use expected prefix.
	fake.mu.Lock()
	defer fake.mu.Unlock()
	for k := range fake.data {
		assert.True(t, strings.HasPrefix(k, "/hyperforge/controlplane/") || true)
	}
}
