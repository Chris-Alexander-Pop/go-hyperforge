// Package ipmi provides an IPMI-over-LAN HTTP gateway adapter for
// provisioning.Provisioner. Prefer redfish for modern BMCs; this adapter
// targets classic IPMI tools exposed via a simple HTTP control plane.
package ipmi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/provisioning"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds IPMI gateway configuration.
type Config struct {
	BaseURL    string `env:"IPMI_GATEWAY_URL"`
	Username   string `env:"IPMI_USERNAME"`
	Password   string `env:"IPMI_PASSWORD"`
	HTTPClient *http.Client
}

// Provisioner implements provisioning.Provisioner via an HTTP IPMI gateway.
type Provisioner struct {
	cfg    Config
	client *http.Client
}

// New creates an IPMI provisioner.
func New(cfg Config) (*Provisioner, error) {
	if cfg.BaseURL == "" {
		return nil, pkgerrors.InvalidArgument("base_url is required", nil)
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Provisioner{cfg: cfg, client: hc}, nil
}

func (p *Provisioner) ProvisionHost(ctx context.Context, hostID string, imageURL string) error {
	return pkgerrors.Unimplemented("ipmi.ProvisionHost not wired; use PXE", nil)
}

func (p *Provisioner) DeprovisionHost(ctx context.Context, hostID string) error {
	return pkgerrors.Unimplemented("ipmi.DeprovisionHost not wired", nil)
}

func (p *Provisioner) GetHostStatus(ctx context.Context, hostID string) (cloud.HostStatus, error) {
	var body struct {
		Power string `json:"power"`
	}
	if err := p.do(ctx, http.MethodGet, "/hosts/"+hostID+"/power", nil, &body); err != nil {
		return cloud.HostStatusUnknown, err
	}
	switch strings.ToLower(body.Power) {
	case "on":
		return cloud.HostStatusReady, nil
	case "off":
		return cloud.HostStatusOffline, nil
	default:
		return cloud.HostStatusUnknown, nil
	}
}

func (p *Provisioner) PowerCycle(ctx context.Context, hostID string) error {
	payload := map[string]string{"action": "cycle"}
	return p.do(ctx, http.MethodPost, "/hosts/"+hostID+"/power", payload, nil)
}

func (p *Provisioner) do(ctx context.Context, method, path string, payload any, out any) error {
	var rdr io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(p.cfg.BaseURL, "/")+path, rdr)
	if err != nil {
		return pkgerrors.Internal("build ipmi request", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.cfg.Username != "" {
		req.SetBasicAuth(p.cfg.Username, p.cfg.Password)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return pkgerrors.Internal("ipmi request failed", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return provisioning.ErrHostNotFound
	}
	if resp.StatusCode >= 400 {
		return pkgerrors.Internal(fmt.Sprintf("ipmi status %d: %s", resp.StatusCode, string(data)), nil)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

var _ provisioning.Provisioner = (*Provisioner)(nil)
