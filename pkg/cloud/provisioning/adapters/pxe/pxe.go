// Package pxe provides a PXE/boot orchestration adapter for provisioning.Provisioner.
//
// It talks to an HTTP control plane that manages DHCP/TFTP boot assignments
// and power/boot triggers. Memory-shaped local state tracks host status;
// HTTPClient + BaseURL make the adapter mockable with httptest.
package pxe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/provisioning"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds PXE orchestrator configuration.
type Config struct {
	// BaseURL is the PXE control-plane root (e.g. http://pxe.local:8080).
	BaseURL string `env:"PXE_BASE_URL"`

	// Username / Password for optional basic auth.
	Username string `env:"PXE_USERNAME"`
	Password string `env:"PXE_PASSWORD"`

	HTTPClient *http.Client
}

// Provisioner implements provisioning.Provisioner over a PXE HTTP API.
type Provisioner struct {
	cfg    Config
	client *http.Client
	mu     *concurrency.SmartRWMutex
	status map[string]cloud.HostStatus
}

// New creates a PXE provisioner.
func New(cfg Config) (*Provisioner, error) {
	if cfg.BaseURL == "" {
		return nil, pkgerrors.InvalidArgument("base_url is required", nil)
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Provisioner{
		cfg:    cfg,
		client: hc,
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "pxe-provisioner"}),
		status: make(map[string]cloud.HostStatus),
	}, nil
}

// SeedHost registers a known host for local status tracking (tests / bootstrap).
func (p *Provisioner) SeedHost(hostID string, status cloud.HostStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status[hostID] = status
}

// ProvisionHost assigns a boot image via PXE and marks the host busy→ready.
func (p *Provisioner) ProvisionHost(ctx context.Context, hostID string, imageURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if hostID == "" || imageURL == "" {
		return pkgerrors.InvalidArgument("hostID and imageURL are required", nil)
	}

	payload := map[string]string{
		"host_id":   hostID,
		"image_url": imageURL,
		"boot":      "pxe",
	}
	if err := p.do(ctx, http.MethodPost, "/hosts/"+hostID+"/provision", payload, nil); err != nil {
		return err
	}

	p.mu.Lock()
	p.status[hostID] = cloud.HostStatusBusy
	p.mu.Unlock()

	// Best-effort status refresh; ignore errors so provision success stands.
	_ = p.refreshStatus(ctx, hostID)
	return nil
}

// DeprovisionHost clears PXE assignment and marks the host offline.
func (p *Provisioner) DeprovisionHost(ctx context.Context, hostID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.do(ctx, http.MethodPost, "/hosts/"+hostID+"/deprovision", map[string]string{"host_id": hostID}, nil); err != nil {
		return err
	}
	p.mu.Lock()
	p.status[hostID] = cloud.HostStatusOffline
	p.mu.Unlock()
	return nil
}

// GetHostStatus returns local status, refreshing from the PXE API when possible.
func (p *Provisioner) GetHostStatus(ctx context.Context, hostID string) (cloud.HostStatus, error) {
	if err := ctx.Err(); err != nil {
		return cloud.HostStatusUnknown, err
	}
	if err := p.refreshStatus(ctx, hostID); err != nil {
		// Fall back to local cache on transport errors only if we know the host.
		p.mu.RLock()
		st, ok := p.status[hostID]
		p.mu.RUnlock()
		if ok {
			return st, nil
		}
		return cloud.HostStatusUnknown, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	st, ok := p.status[hostID]
	if !ok {
		return cloud.HostStatusUnknown, provisioning.ErrHostNotFound
	}
	return st, nil
}

// PowerCycle triggers a PXE reboot / cold boot via the control plane.
func (p *Provisioner) PowerCycle(ctx context.Context, hostID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.do(ctx, http.MethodPost, "/hosts/"+hostID+"/power", map[string]string{"action": "cycle"}, nil)
}

func (p *Provisioner) refreshStatus(ctx context.Context, hostID string) error {
	var body struct {
		Status string `json:"status"`
		State  string `json:"state"`
	}
	if err := p.do(ctx, http.MethodGet, "/hosts/"+hostID+"/status", nil, &body); err != nil {
		return err
	}
	st := parseStatus(body.Status, body.State)
	p.mu.Lock()
	p.status[hostID] = st
	p.mu.Unlock()
	return nil
}

func parseStatus(status, state string) cloud.HostStatus {
	s := strings.ToLower(status)
	if s == "" {
		s = strings.ToLower(state)
	}
	switch s {
	case "ready", "available", "on", "provisioned":
		return cloud.HostStatusReady
	case "busy", "provisioning", "booting":
		return cloud.HostStatusBusy
	case "offline", "off", "deprovisioned":
		return cloud.HostStatusOffline
	case "maintenance":
		return cloud.HostStatusMaintenance
	default:
		return cloud.HostStatusUnknown
	}
}

func (p *Provisioner) do(ctx context.Context, method, path string, payload any, out any) error {
	var rdr io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return pkgerrors.Internal("marshal pxe payload", err)
		}
		rdr = bytes.NewReader(b)
	}
	url := strings.TrimRight(p.cfg.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return pkgerrors.Internal("build pxe request", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.cfg.Username != "" {
		req.SetBasicAuth(p.cfg.Username, p.cfg.Password)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return pkgerrors.Internal("pxe request failed", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return provisioning.ErrHostNotFound
	}
	if resp.StatusCode >= 400 {
		return pkgerrors.Internal(fmt.Sprintf("pxe status %d: %s", resp.StatusCode, string(data)), nil)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return pkgerrors.Internal("decode pxe response", err)
		}
	}
	return nil
}

var _ provisioning.Provisioner = (*Provisioner)(nil)
