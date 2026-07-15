package redfish

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

// Config holds Redfish BMC configuration.
type Config struct {
	// BaseURL is the BMC root, e.g. https://bmc.example/redfish/v1
	BaseURL  string `env:"REDFISH_BASE_URL"`
	Username string `env:"REDFISH_USERNAME"`
	Password string `env:"REDFISH_PASSWORD"`
	// HostSystems maps logical host IDs to Redfish Systems resource paths
	// (relative to BaseURL), e.g. "/Systems/1".
	HostSystems map[string]string
	HTTPClient  *http.Client
}

// Provisioner implements provisioning.Provisioner over Redfish.
type Provisioner struct {
	cfg    Config
	client *http.Client
}

// New creates a Redfish provisioner.
func New(cfg Config) (*Provisioner, error) {
	if cfg.BaseURL == "" {
		return nil, pkgerrors.InvalidArgument("base_url is required", nil)
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.HostSystems == nil {
		cfg.HostSystems = map[string]string{}
	}
	return &Provisioner{cfg: cfg, client: hc}, nil
}

func (p *Provisioner) ProvisionHost(ctx context.Context, hostID string, imageURL string) error {
	return pkgerrors.Unimplemented("redfish.ProvisionHost (OS imaging) not wired; use PXE adapter", nil)
}

func (p *Provisioner) DeprovisionHost(ctx context.Context, hostID string) error {
	return pkgerrors.Unimplemented("redfish.DeprovisionHost not wired", nil)
}

func (p *Provisioner) GetHostStatus(ctx context.Context, hostID string) (cloud.HostStatus, error) {
	sys, err := p.systemPath(hostID)
	if err != nil {
		return cloud.HostStatusUnknown, err
	}
	var body struct {
		PowerState string `json:"PowerState"`
		Status     struct {
			State string `json:"State"`
		} `json:"Status"`
	}
	if err := p.doJSON(ctx, http.MethodGet, sys, nil, &body); err != nil {
		return cloud.HostStatusUnknown, err
	}
	switch strings.ToLower(body.PowerState) {
	case "on":
		return cloud.HostStatusReady, nil
	case "off", "poweredoff":
		return cloud.HostStatusOffline, nil
	default:
		if strings.EqualFold(body.Status.State, "Enabled") {
			return cloud.HostStatusReady, nil
		}
		return cloud.HostStatusUnknown, nil
	}
}

func (p *Provisioner) PowerCycle(ctx context.Context, hostID string) error {
	sys, err := p.systemPath(hostID)
	if err != nil {
		return err
	}
	payload := map[string]string{"ResetType": "ForceRestart"}
	return p.doJSON(ctx, http.MethodPost, sys+"/Actions/ComputerSystem.Reset", payload, nil)
}

func (p *Provisioner) systemPath(hostID string) (string, error) {
	if path, ok := p.cfg.HostSystems[hostID]; ok && path != "" {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return path, nil
	}
	// Default convention: /Systems/{hostID}
	return "/Systems/" + hostID, nil
}

func (p *Provisioner) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var rdr io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return pkgerrors.Internal("marshal redfish payload", err)
		}
		rdr = strings.NewReader(string(b))
	}
	url := strings.TrimRight(p.cfg.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return pkgerrors.Internal("build redfish request", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.cfg.Username != "" {
		req.SetBasicAuth(p.cfg.Username, p.cfg.Password)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return pkgerrors.Internal("redfish request failed", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return provisioning.ErrHostNotFound
	}
	if resp.StatusCode >= 400 {
		return pkgerrors.Internal(fmt.Sprintf("redfish status %d: %s", resp.StatusCode, string(data)), nil)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return pkgerrors.Internal("decode redfish response", err)
		}
	}
	return nil
}

var _ provisioning.Provisioner = (*Provisioner)(nil)
