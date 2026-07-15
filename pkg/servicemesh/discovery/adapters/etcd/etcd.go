package etcd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	"github.com/google/uuid"
)

// Config configures the etcd HTTP discovery adapter.
type Config struct {
	// Address is the etcd HTTP base URL (gRPC-gateway), e.g. http://127.0.0.1:2379.
	Address string `env:"ETCD_ADDRESS" env-default:"http://127.0.0.1:2379"`

	// Prefix is the key namespace (default /hyperforge/discovery).
	Prefix string `env:"ETCD_DISCOVERY_PREFIX" env-default:"/hyperforge/discovery"`

	// Namespace is stamped onto registered services when not set elsewhere.
	Namespace string `env:"DISCOVERY_NAMESPACE" env-default:"default"`

	// HTTPClient overrides the default client (15s timeout).
	HTTPClient *http.Client

	// WatchInterval is the poll interval for Watch (default 1s).
	// etcd watch streams are not used; this is a thin KV poller.
	WatchInterval time.Duration
}

// Registry talks to etcd over the v3 HTTP KV API.
type Registry struct {
	base   string
	prefix string
	ns     string
	client *http.Client
	watch  time.Duration

	mu     *concurrency.SmartMutex
	closed bool
}

var _ discovery.ServiceRegistry = (*Registry)(nil)

// New creates an etcd HTTP service registry.
func New(cfg Config) (*Registry, error) {
	addr := strings.TrimRight(strings.TrimSpace(cfg.Address), "/")
	if addr == "" {
		return nil, discovery.ErrInvalidService
	}
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "/hyperforge/discovery"
	}
	prefix = strings.TrimRight(prefix, "/")
	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	watch := cfg.WatchInterval
	if watch <= 0 {
		watch = time.Second
	}
	return &Registry{
		base:   addr,
		prefix: prefix,
		ns:     ns,
		client: client,
		watch:  watch,
		mu:     concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "discovery-etcd"}),
	}, nil
}

func (r *Registry) serviceKey(name, id string) string {
	return r.prefix + "/services/" + name + "/" + id
}

func (r *Registry) namePrefix(name string) string {
	return r.prefix + "/services/" + name + "/"
}

func (r *Registry) allPrefix() string {
	return r.prefix + "/services/"
}

func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func decodeB64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

type kvPutReq struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type kvRangeReq struct {
	Key      string `json:"key"`
	RangeEnd string `json:"range_end,omitempty"`
}

type kvDeleteReq struct {
	Key string `json:"key"`
}

type kvResp struct {
	KVs []kv `json:"kvs"`
}

type kv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func rangeEnd(prefix string) string {
	b := []byte(prefix)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] < 0xff {
			b[i]++
			return string(b[:i+1])
		}
	}
	return prefix + "\x00"
}

func (r *Registry) doJSON(ctx context.Context, path string, body any, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return errors.Internal("failed to encode etcd request", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.base+path, &buf)
	if err != nil {
		return errors.InvalidArgument("failed to build etcd request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.Unavailable("etcd unreachable", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return errors.Internal("failed to read etcd response", err)
	}
	if resp.StatusCode >= 300 {
		return errors.Unavailable(fmt.Sprintf("etcd returned %d: %s", resp.StatusCode, string(data)), nil)
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return errors.Internal("failed to decode etcd response", err)
	}
	return nil
}

func (r *Registry) put(ctx context.Context, key string, value []byte) error {
	return r.doJSON(ctx, "/v3/kv/put", kvPutReq{
		Key:   b64(key),
		Value: base64.StdEncoding.EncodeToString(value),
	}, &kvResp{})
}

func (r *Registry) deleteKey(ctx context.Context, key string) error {
	return r.doJSON(ctx, "/v3/kv/deleterange", kvDeleteReq{Key: b64(key)}, &kvResp{})
}

func (r *Registry) listPrefix(ctx context.Context, prefix string) ([][]byte, error) {
	var resp kvResp
	if err := r.doJSON(ctx, "/v3/kv/range", kvRangeReq{
		Key:      b64(prefix),
		RangeEnd: b64(rangeEnd(prefix)),
	}, &resp); err != nil {
		return nil, err
	}
	out := make([][]byte, 0, len(resp.KVs))
	for _, item := range resp.KVs {
		raw, err := decodeB64(item.Value)
		if err != nil {
			return nil, errors.Internal("invalid etcd value encoding", err)
		}
		out = append(out, raw)
	}
	return out, nil
}

func (r *Registry) decodeService(raw []byte) (*discovery.Service, error) {
	var svc discovery.Service
	if err := json.Unmarshal(raw, &svc); err != nil {
		return nil, errors.Internal("failed to decode service", err)
	}
	return &svc, nil
}

func (r *Registry) errIfClosed() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return discovery.ErrWatchClosed
	}
	return nil
}

// Register stores a service instance under /prefix/services/{name}/{id}.
func (r *Registry) Register(ctx context.Context, opts discovery.RegisterOptions) (*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if opts.Name == "" {
		return nil, discovery.ErrInvalidService
	}
	id := opts.ID
	if id == "" {
		id = uuid.NewString()
	}
	weight := opts.Weight
	if weight <= 0 {
		weight = 1
	}
	now := time.Now().UTC()
	svc := &discovery.Service{
		ID:            id,
		Name:          opts.Name,
		Address:       opts.Address,
		Port:          opts.Port,
		Tags:          opts.Tags,
		Metadata:      opts.Metadata,
		Health:        discovery.HealthStatusPassing,
		Namespace:     r.ns,
		Weight:        weight,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}
	if svc.Metadata == nil {
		svc.Metadata = map[string]string{}
	}
	raw, err := json.Marshal(svc)
	if err != nil {
		return nil, errors.Internal("failed to encode service", err)
	}
	if err := r.put(ctx, r.serviceKey(opts.Name, id), raw); err != nil {
		return nil, err
	}
	return svc, nil
}

// Deregister removes a service instance. Looks up by scanning when name is unknown.
func (r *Registry) Deregister(ctx context.Context, serviceID string) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	if serviceID == "" {
		return discovery.ErrInvalidService
	}
	svc, err := r.Get(ctx, serviceID)
	if err != nil {
		return err
	}
	return r.deleteKey(ctx, r.serviceKey(svc.Name, serviceID))
}

// Lookup returns instances for a service name.
func (r *Registry) Lookup(ctx context.Context, serviceName string, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceName == "" {
		return nil, discovery.ErrInvalidService
	}
	raws, err := r.listPrefix(ctx, r.namePrefix(serviceName))
	if err != nil {
		return nil, err
	}
	return r.filterServices(raws, opts), nil
}

// Get retrieves a service by ID (prefix scan).
func (r *Registry) Get(ctx context.Context, serviceID string) (*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceID == "" {
		return nil, discovery.ErrInvalidService
	}
	raws, err := r.listPrefix(ctx, r.allPrefix())
	if err != nil {
		return nil, err
	}
	for _, raw := range raws {
		svc, err := r.decodeService(raw)
		if err != nil {
			continue
		}
		if svc.ID == serviceID {
			return svc, nil
		}
	}
	return nil, discovery.ErrServiceNotFound
}

// List returns all registered services.
func (r *Registry) List(ctx context.Context, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	raws, err := r.listPrefix(ctx, r.allPrefix())
	if err != nil {
		return nil, err
	}
	return r.filterServices(raws, opts), nil
}

func (r *Registry) filterServices(raws [][]byte, opts discovery.QueryOptions) []*discovery.Service {
	out := make([]*discovery.Service, 0, len(raws))
	for _, raw := range raws {
		svc, err := r.decodeService(raw)
		if err != nil {
			continue
		}
		if opts.Tag != "" && !containsTag(svc.Tags, opts.Tag) {
			continue
		}
		if opts.Namespace != "" && svc.Namespace != opts.Namespace {
			continue
		}
		if opts.HealthyOnly && svc.Health != discovery.HealthStatusPassing {
			continue
		}
		out = append(out, svc)
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out
}

// Watch polls etcd for changes to a service name.
func (r *Registry) Watch(ctx context.Context, serviceName string) (<-chan []*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceName == "" {
		return nil, discovery.ErrInvalidService
	}
	ch := make(chan []*discovery.Service, 4)
	go func() {
		defer close(ch)
		var last string
		for {
			if ctx.Err() != nil {
				return
			}
			r.mu.Lock()
			closed := r.closed
			r.mu.Unlock()
			if closed {
				return
			}
			svcs, err := r.Lookup(ctx, serviceName, discovery.QueryOptions{HealthyOnly: false})
			if err == nil {
				raw, _ := json.Marshal(svcs)
				sig := string(raw)
				if sig != last {
					last = sig
					select {
					case ch <- svcs:
					case <-ctx.Done():
						return
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(r.watch):
			}
		}
	}()
	return ch, nil
}

// Heartbeat refreshes LastHeartbeat on the stored service.
func (r *Registry) Heartbeat(ctx context.Context, serviceID string) error {
	return r.touch(ctx, serviceID, "")
}

// UpdateHealth updates the health status of a service.
func (r *Registry) UpdateHealth(ctx context.Context, serviceID string, status discovery.HealthStatus) error {
	return r.touch(ctx, serviceID, status)
}

func (r *Registry) touch(ctx context.Context, serviceID string, status discovery.HealthStatus) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	if serviceID == "" {
		return discovery.ErrInvalidService
	}
	svc, err := r.Get(ctx, serviceID)
	if err != nil {
		return err
	}
	svc.LastHeartbeat = time.Now().UTC()
	if status != "" {
		svc.Health = status
	}
	raw, err := json.Marshal(svc)
	if err != nil {
		return errors.Internal("failed to encode service", err)
	}
	return r.put(ctx, r.serviceKey(svc.Name, svc.ID), raw)
}

// Close marks the registry closed and stops watches.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
