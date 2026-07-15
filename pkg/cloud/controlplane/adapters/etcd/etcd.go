// Package etcd provides a ControlPlane backed by the etcd v3 HTTP JSON API.
//
// Hosts and instances are stored as JSON values under configurable key prefixes.
// Tests use httptest; production points Address at an etcd gRPC-gateway
// (e.g. http://127.0.0.1:2379).
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

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Ensure compile-time compliance.
var _ controlplane.ControlPlane = (*ControlPlane)(nil)

// Config configures the etcd HTTP control-plane adapter.
type Config struct {
	// Address is the etcd HTTP base URL (gRPC-gateway), e.g. http://127.0.0.1:2379.
	Address string `env:"ETCD_ADDRESS" env-default:"http://127.0.0.1:2379"`

	// Prefix is the key namespace (default /hyperforge/controlplane).
	Prefix string `env:"ETCD_CONTROLPLANE_PREFIX" env-default:"/hyperforge/controlplane"`

	// HTTPClient overrides the default client.
	HTTPClient *http.Client
}

// ControlPlane persists host inventory (and instances) in etcd via HTTP.
type ControlPlane struct {
	base   string
	prefix string
	client *http.Client
}

// New creates an etcd-backed control plane.
func New(cfg Config) (*ControlPlane, error) {
	addr := strings.TrimRight(strings.TrimSpace(cfg.Address), "/")
	if addr == "" {
		return nil, pkgerrors.InvalidArgument("etcd address is required", nil)
	}
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "/hyperforge/controlplane"
	}
	prefix = strings.TrimRight(prefix, "/")
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &ControlPlane{base: addr, prefix: prefix, client: client}, nil
}

func (c *ControlPlane) hostKey(id string) string {
	return c.prefix + "/hosts/" + id
}

func (c *ControlPlane) instanceKey(id string) string {
	return c.prefix + "/instances/" + id
}

func (c *ControlPlane) hostsPrefix() string {
	return c.prefix + "/hosts/"
}

func (c *ControlPlane) instancesPrefix() string {
	return c.prefix + "/instances/"
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
	Key      string `json:"key"`
	RangeEnd string `json:"range_end,omitempty"`
}

type kvResp struct {
	KVs   []kv   `json:"kvs"`
	Count string `json:"count"`
}

type kv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *ControlPlane) doJSON(ctx context.Context, path string, body any, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return pkgerrors.Internal("failed to encode etcd request", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, &buf)
	if err != nil {
		return pkgerrors.Internal("failed to create etcd request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return pkgerrors.Unavailable("etcd request failed", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return pkgerrors.Internal("failed to read etcd response", err)
	}
	if resp.StatusCode >= 300 {
		return pkgerrors.Unavailable(fmt.Sprintf("etcd returned %d: %s", resp.StatusCode, string(data)), nil)
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return pkgerrors.Internal("failed to decode etcd response", err)
	}
	return nil
}

func (c *ControlPlane) put(ctx context.Context, key string, value []byte) error {
	return c.doJSON(ctx, "/v3/kv/put", kvPutReq{
		Key:   b64(key),
		Value: base64.StdEncoding.EncodeToString(value),
	}, &kvResp{})
}

func (c *ControlPlane) get(ctx context.Context, key string) ([]byte, bool, error) {
	var resp kvResp
	if err := c.doJSON(ctx, "/v3/kv/range", kvRangeReq{Key: b64(key)}, &resp); err != nil {
		return nil, false, err
	}
	if len(resp.KVs) == 0 {
		return nil, false, nil
	}
	raw, err := decodeB64(resp.KVs[0].Value)
	if err != nil {
		return nil, false, pkgerrors.Internal("invalid etcd value encoding", err)
	}
	return raw, true, nil
}

func (c *ControlPlane) deleteKey(ctx context.Context, key string) error {
	return c.doJSON(ctx, "/v3/kv/deleterange", kvDeleteReq{Key: b64(key)}, &kvResp{})
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

func (c *ControlPlane) listPrefix(ctx context.Context, prefix string) ([][]byte, error) {
	var resp kvResp
	if err := c.doJSON(ctx, "/v3/kv/range", kvRangeReq{
		Key:      b64(prefix),
		RangeEnd: b64(rangeEnd(prefix)),
	}, &resp); err != nil {
		return nil, err
	}
	out := make([][]byte, 0, len(resp.KVs))
	for _, item := range resp.KVs {
		raw, err := decodeB64(item.Value)
		if err != nil {
			return nil, pkgerrors.Internal("invalid etcd value encoding", err)
		}
		out = append(out, raw)
	}
	return out, nil
}

func (c *ControlPlane) loadHost(ctx context.Context, hostID string) (cloud.Host, error) {
	raw, ok, err := c.get(ctx, c.hostKey(hostID))
	if err != nil {
		return cloud.Host{}, err
	}
	if !ok {
		return cloud.Host{}, controlplane.ErrHostNotFound
	}
	var h cloud.Host
	if err := json.Unmarshal(raw, &h); err != nil {
		return cloud.Host{}, pkgerrors.Internal("failed to decode host", err)
	}
	return h, nil
}

func (c *ControlPlane) saveHost(ctx context.Context, h cloud.Host) error {
	raw, err := json.Marshal(h)
	if err != nil {
		return pkgerrors.Internal("failed to encode host", err)
	}
	return c.put(ctx, c.hostKey(h.ID), raw)
}

func (c *ControlPlane) loadInstance(ctx context.Context, id string) (controlplane.Instance, error) {
	raw, ok, err := c.get(ctx, c.instanceKey(id))
	if err != nil {
		return controlplane.Instance{}, err
	}
	if !ok {
		return controlplane.Instance{}, controlplane.ErrInstanceNotFound
	}
	var inst controlplane.Instance
	if err := json.Unmarshal(raw, &inst); err != nil {
		return controlplane.Instance{}, pkgerrors.Internal("failed to decode instance", err)
	}
	return inst, nil
}

func (c *ControlPlane) saveInstance(ctx context.Context, inst controlplane.Instance) error {
	raw, err := json.Marshal(inst)
	if err != nil {
		return pkgerrors.Internal("failed to encode instance", err)
	}
	return c.put(ctx, c.instanceKey(inst.ID), raw)
}

// RegisterHost adds a host to etcd inventory.
func (c *ControlPlane) RegisterHost(ctx context.Context, host cloud.Host) error {
	if host.ID == "" {
		return pkgerrors.InvalidArgument("host id is required", nil)
	}
	if _, err := c.loadHost(ctx, host.ID); err == nil {
		return controlplane.ErrHostAlreadyRegistered
	} else if err != controlplane.ErrHostNotFound {
		return err
	}
	if host.Available.VCPUs == 0 && host.Available.MemoryMB == 0 && host.Available.DiskGB == 0 {
		host.Available = host.Capacity
	}
	return c.saveHost(ctx, host)
}

// DeregisterHost removes a host when it has no bound instances.
func (c *ControlPlane) DeregisterHost(ctx context.Context, hostID string) error {
	if _, err := c.loadHost(ctx, hostID); err != nil {
		return err
	}
	insts, err := c.ListInstances(ctx, controlplane.ListInstancesOptions{HostID: hostID})
	if err != nil {
		return err
	}
	if len(insts) > 0 {
		return controlplane.ErrHostHasInstances
	}
	return c.deleteKey(ctx, c.hostKey(hostID))
}

// UpdateHostStatus updates a registered host's status.
func (c *ControlPlane) UpdateHostStatus(ctx context.Context, hostID string, status cloud.HostStatus) error {
	h, err := c.loadHost(ctx, hostID)
	if err != nil {
		return err
	}
	h.Status = status
	return c.saveHost(ctx, h)
}

// GetHost retrieves a host by ID.
func (c *ControlPlane) GetHost(ctx context.Context, hostID string) (*cloud.Host, error) {
	h, err := c.loadHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// ListHosts returns all registered hosts.
func (c *ControlPlane) ListHosts(ctx context.Context) ([]cloud.Host, error) {
	raws, err := c.listPrefix(ctx, c.hostsPrefix())
	if err != nil {
		return nil, err
	}
	out := make([]cloud.Host, 0, len(raws))
	for _, raw := range raws {
		var h cloud.Host
		if err := json.Unmarshal(raw, &h); err != nil {
			return nil, pkgerrors.Internal("failed to decode host", err)
		}
		out = append(out, h)
	}
	return out, nil
}

// CreateInstance registers an instance, optionally binding to a host.
func (c *ControlPlane) CreateInstance(ctx context.Context, req controlplane.CreateInstanceRequest) (*controlplane.Instance, error) {
	if req.Name == "" {
		return nil, pkgerrors.InvalidArgument("instance name is required", nil)
	}
	inst := controlplane.Instance{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Status:    cloud.InstanceStatusPending,
		Resources: req.Resources,
		Image:     req.Image,
		Tags:      req.Tags,
		CreatedAt: time.Now().UTC(),
	}
	if req.HostID != "" {
		if err := c.bind(ctx, &inst, req.HostID); err != nil {
			return nil, err
		}
		inst.Status = cloud.InstanceStatusProvisioning
	}
	if err := c.saveInstance(ctx, inst); err != nil {
		return nil, err
	}
	out := inst
	return &out, nil
}

// BindInstance assigns an unbound instance to a host.
func (c *ControlPlane) BindInstance(ctx context.Context, instanceID, hostID string) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.HostID != "" {
		return controlplane.ErrInstanceAlreadyBound
	}
	if err := c.bind(ctx, &inst, hostID); err != nil {
		return err
	}
	if inst.Status == cloud.InstanceStatusPending {
		inst.Status = cloud.InstanceStatusProvisioning
	}
	return c.saveInstance(ctx, inst)
}

// UnbindInstance detaches an instance from its host.
func (c *ControlPlane) UnbindInstance(ctx context.Context, instanceID string) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.HostID == "" {
		return controlplane.ErrInstanceNotBound
	}
	if err := c.release(ctx, inst); err != nil {
		return err
	}
	inst.HostID = ""
	return c.saveInstance(ctx, inst)
}

// GetInstance retrieves an instance by ID.
func (c *ControlPlane) GetInstance(ctx context.Context, instanceID string) (*controlplane.Instance, error) {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// ListInstances returns instances filtered by options.
func (c *ControlPlane) ListInstances(ctx context.Context, opts controlplane.ListInstancesOptions) ([]controlplane.Instance, error) {
	raws, err := c.listPrefix(ctx, c.instancesPrefix())
	if err != nil {
		return nil, err
	}
	out := make([]controlplane.Instance, 0, len(raws))
	for _, raw := range raws {
		var inst controlplane.Instance
		if err := json.Unmarshal(raw, &inst); err != nil {
			return nil, pkgerrors.Internal("failed to decode instance", err)
		}
		if opts.HostID != "" && inst.HostID != opts.HostID {
			continue
		}
		if opts.Status != "" && inst.Status != opts.Status {
			continue
		}
		out = append(out, inst)
	}
	return out, nil
}

// DeleteInstance removes an instance and releases capacity.
func (c *ControlPlane) DeleteInstance(ctx context.Context, instanceID string) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.HostID != "" {
		if err := c.release(ctx, inst); err != nil {
			return err
		}
	}
	return c.deleteKey(ctx, c.instanceKey(instanceID))
}

// UpdateInstanceStatus updates instance lifecycle status.
func (c *ControlPlane) UpdateInstanceStatus(ctx context.Context, instanceID string, status cloud.InstanceStatus) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	inst.Status = status
	return c.saveInstance(ctx, inst)
}

func (c *ControlPlane) bind(ctx context.Context, inst *controlplane.Instance, hostID string) error {
	host, err := c.loadHost(ctx, hostID)
	if err != nil {
		return err
	}
	if host.Status != cloud.HostStatusReady && host.Status != cloud.HostStatusBusy {
		return controlplane.ErrHostNotReady
	}
	res := inst.Resources
	if host.Available.VCPUs < res.VCPUs || host.Available.MemoryMB < res.MemoryMB || host.Available.DiskGB < res.DiskGB {
		return controlplane.ErrHostCapacityExhausted
	}
	host.Available.VCPUs -= res.VCPUs
	host.Available.MemoryMB -= res.MemoryMB
	host.Available.DiskGB -= res.DiskGB
	host.Available.GPUs -= res.GPUs
	if host.Available.VCPUs == 0 || host.Available.MemoryMB == 0 {
		host.Status = cloud.HostStatusBusy
	}
	if err := c.saveHost(ctx, host); err != nil {
		return err
	}
	inst.HostID = hostID
	return nil
}

func (c *ControlPlane) release(ctx context.Context, inst controlplane.Instance) error {
	host, err := c.loadHost(ctx, inst.HostID)
	if err != nil {
		if err == controlplane.ErrHostNotFound {
			return nil
		}
		return err
	}
	host.Available.VCPUs += inst.Resources.VCPUs
	host.Available.MemoryMB += inst.Resources.MemoryMB
	host.Available.DiskGB += inst.Resources.DiskGB
	host.Available.GPUs += inst.Resources.GPUs
	if host.Status == cloud.HostStatusBusy {
		host.Status = cloud.HostStatusReady
	}
	return c.saveHost(ctx, host)
}
