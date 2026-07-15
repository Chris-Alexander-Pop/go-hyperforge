package libvirt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/hypervisor"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds remote libvirt gateway configuration.
type Config struct {
	// BaseURL is the JSON/RPC gateway root (e.g. http://127.0.0.1:16509).
	BaseURL string `env:"HYPERVISOR_LIBVIRT_URL"`
	// URI is informational (qemu:///system); forwarded in create payloads.
	URI        string `env:"HYPERVISOR_URI" env-default:"qemu:///system"`
	HTTPClient *http.Client
}

// Hypervisor talks to a remote libvirt JSON gateway.
type Hypervisor struct {
	base   string
	uri    string
	client *http.Client
}

// New creates a remote libvirt hypervisor client.
func New(cfg Config) (*Hypervisor, error) {
	if cfg.BaseURL == "" {
		return nil, pkgerrors.InvalidArgument("base_url is required for remote libvirt", nil)
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	uri := cfg.URI
	if uri == "" {
		uri = "qemu:///system"
	}
	return &Hypervisor{base: strings.TrimRight(cfg.BaseURL, "/"), uri: uri, client: hc}, nil
}

type vmDTO struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Status    cloud.InstanceStatus `json:"status"`
	Spec      hypervisor.VMSpec    `json:"spec"`
	CreatedAt time.Time            `json:"created_at"`
	IPAddress string               `json:"ip_address,omitempty"`
}

type createReq struct {
	URI  string            `json:"uri"`
	Spec hypervisor.VMSpec `json:"spec"`
}

type createResp struct {
	ID string `json:"id"`
}

type errResp struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func (h *Hypervisor) CreateVM(ctx context.Context, spec hypervisor.VMSpec) (string, error) {
	if spec.Name == "" {
		return "", pkgerrors.InvalidArgument("name is required", nil)
	}
	body, _ := json.Marshal(createReq{URI: h.uri, Spec: spec})
	var out createResp
	if err := h.do(ctx, http.MethodPost, "/v1/vms", body, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", pkgerrors.Internal("libvirt gateway returned empty id", nil)
	}
	return out.ID, nil
}

func (h *Hypervisor) StartVM(ctx context.Context, vmID string) error {
	return h.do(ctx, http.MethodPost, "/v1/vms/"+vmID+"/start", nil, nil)
}

func (h *Hypervisor) StopVM(ctx context.Context, vmID string) error {
	return h.do(ctx, http.MethodPost, "/v1/vms/"+vmID+"/stop", nil, nil)
}

func (h *Hypervisor) DeleteVM(ctx context.Context, vmID string) error {
	return h.do(ctx, http.MethodDelete, "/v1/vms/"+vmID, nil, nil)
}

func (h *Hypervisor) GetVMStatus(ctx context.Context, vmID string) (cloud.InstanceStatus, error) {
	var dto vmDTO
	if err := h.do(ctx, http.MethodGet, "/v1/vms/"+vmID, nil, &dto); err != nil {
		return cloud.InstanceStatusUnknown, err
	}
	if dto.Status == "" {
		return cloud.InstanceStatusUnknown, nil
	}
	return dto.Status, nil
}

func (h *Hypervisor) ListVMs(ctx context.Context) ([]hypervisor.VM, error) {
	var dtos []vmDTO
	if err := h.do(ctx, http.MethodGet, "/v1/vms", nil, &dtos); err != nil {
		return nil, err
	}
	out := make([]hypervisor.VM, 0, len(dtos))
	for _, d := range dtos {
		out = append(out, hypervisor.VM{
			ID: d.ID, Name: d.Name, Status: d.Status, Spec: d.Spec,
			CreatedAt: d.CreatedAt, IPAddress: d.IPAddress,
		})
	}
	return out, nil
}

func (h *Hypervisor) do(ctx context.Context, method, path string, body []byte, out any) error {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.base+path, rdr)
	if err != nil {
		return pkgerrors.Internal("failed to build libvirt request", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return pkgerrors.Internal("libvirt gateway request failed", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return hypervisor.ErrVMNotFound
	}
	if resp.StatusCode == http.StatusConflict {
		return hypervisor.ErrVMAlreadyExists
	}
	if resp.StatusCode >= 400 {
		var er errResp
		_ = json.Unmarshal(data, &er)
		msg := er.Error
		if msg == "" {
			msg = fmt.Sprintf("libvirt gateway status %d", resp.StatusCode)
		}
		return pkgerrors.Internal(msg, nil)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return pkgerrors.Internal("failed to decode libvirt response", err)
		}
	}
	return nil
}

var _ hypervisor.Hypervisor = (*Hypervisor)(nil)
