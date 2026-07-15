package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

const defaultAPIBase = "https://api.cloudflare.com/client/v4"

// Config configures the Cloudflare WAF (IP access rules) adapter.
type Config struct {
	// APIToken is a Cloudflare API token with Zone Firewall edit permissions.
	APIToken string `env:"CLOUDFLARE_API_TOKEN" validate:"required"`

	// ZoneID is the Cloudflare zone to manage rules for.
	ZoneID string `env:"CLOUDFLARE_ZONE_ID" validate:"required"`

	// BaseURL overrides the Cloudflare API base (tests).
	BaseURL string

	// HTTPClient is optional; defaults to a 15s timeout client.
	HTTPClient *http.Client

	// Retrier wraps API calls; nil uses resilience.DefaultRetryConfig.
	Retrier resilience.Retrier
}

// Validate checks required Config fields.
func (c Config) Validate() error {
	if err := validator.New().ValidateStruct(context.Background(), c); err != nil {
		if errors.IsCode(err, errors.CodeInvalidArgument) {
			return err
		}
		return errors.New(waf.CodeInvalidRule, "invalid cloudflare waf config", err)
	}
	return nil
}

// Manager implements waf.Manager via Cloudflare IP access rules.
type Manager struct {
	token   string
	zoneID  string
	baseURL string
	client  *http.Client
	retrier resilience.Retrier
}

// Ensure Manager implements waf.Manager.
var _ waf.Manager = (*Manager)(nil)

type apiResponse struct {
	Success bool            `json:"success"`
	Errors  []apiError      `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type accessRule struct {
	ID            string `json:"id"`
	Mode          string `json:"mode"`
	Notes         string `json:"notes"`
	CreatedOn     string `json:"created_on"`
	Configuration struct {
		Target string `json:"target"`
		Value  string `json:"value"`
	} `json:"configuration"`
}

// New creates a Cloudflare WAF manager.
func New(cfg Config) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultAPIBase
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	retrier := cfg.Retrier
	if retrier == nil {
		retrier = resilience.NewRetrier(resilience.DefaultRetryConfig())
	}
	return &Manager{
		token:   cfg.APIToken,
		zoneID:  cfg.ZoneID,
		baseURL: strings.TrimRight(base, "/"),
		client:  client,
		retrier: retrier,
	}, nil
}

func (m *Manager) rulesURL() string {
	return fmt.Sprintf("%s/zones/%s/firewall/access_rules/rules", m.baseURL, url.PathEscape(m.zoneID))
}

func (m *Manager) do(ctx context.Context, method, endpoint string, body io.Reader) (*apiResponse, int, error) {
	var (
		parsed apiResponse
		status int
	)
	err := m.retrier.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
		if err != nil {
			return errors.New(waf.CodeInvalidRule, "failed to build cloudflare request", err)
		}
		req.Header.Set("Authorization", "Bearer "+m.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := m.client.Do(req)
		if err != nil {
			return errors.New(waf.CodeUnavailable, "cloudflare api unreachable", err)
		}
		defer resp.Body.Close()

		raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return errors.New(waf.CodeUnavailable, "failed to read cloudflare response", err)
		}
		status = resp.StatusCode
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			return errors.New(waf.CodeUnavailable, "cloudflare auth failed", nil)
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return errors.New(waf.CodeUnavailable, "invalid cloudflare response", err)
		}
		return nil
	})
	return &parsed, status, err
}

// BlockIP creates a Cloudflare IP block access rule.
func (m *Manager) BlockIP(ctx context.Context, ip, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ip == "" {
		return waf.ErrInvalidRule
	}

	payload, err := json.Marshal(map[string]interface{}{
		"mode":  "block",
		"notes": reason,
		"configuration": map[string]string{
			"target": "ip",
			"value":  ip,
		},
	})
	if err != nil {
		return errors.New(waf.CodeInvalidRule, "failed to marshal cloudflare payload", err)
	}

	resp, status, err := m.do(ctx, http.MethodPost, m.rulesURL(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 || !resp.Success {
		msg := "cloudflare block ip failed"
		if len(resp.Errors) > 0 {
			msg = resp.Errors[0].Message
		}
		return errors.New(waf.CodeUnavailable, msg, nil)
	}
	return nil
}

// AllowIP removes any block access rule matching the IP.
func (m *Manager) AllowIP(ctx context.Context, ip string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ip == "" {
		return waf.ErrInvalidRule
	}

	rules, err := m.listRules(ctx)
	if err != nil {
		return err
	}

	found := false
	for _, r := range rules {
		if r.Configuration.Target == "ip" && r.Configuration.Value == ip && r.Mode == "block" {
			found = true
			delURL := m.rulesURL() + "/" + url.PathEscape(r.ID)
			resp, status, err := m.do(ctx, http.MethodDelete, delURL, nil)
			if err != nil {
				return err
			}
			if status < 200 || status >= 300 || !resp.Success {
				return errors.New(waf.CodeUnavailable, "cloudflare delete rule failed", nil)
			}
		}
	}
	if !found {
		return waf.ErrNotFound
	}
	return nil
}

// GetRules lists Cloudflare IP access rules for the zone.
func (m *Manager) GetRules(ctx context.Context) ([]waf.Rule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rules, err := m.listRules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]waf.Rule, 0, len(rules))
	for _, r := range rules {
		out = append(out, waf.Rule{
			ID:     r.ID,
			IP:     r.Configuration.Value,
			Action: r.Mode,
			Reason: r.Notes,
		})
	}
	return out, nil
}

func (m *Manager) listRules(ctx context.Context) ([]accessRule, error) {
	resp, status, err := m.do(ctx, http.MethodGet, m.rulesURL(), nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 || !resp.Success {
		return nil, errors.New(waf.CodeUnavailable, "cloudflare list rules failed", nil)
	}
	var rules []accessRule
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return rules, nil
	}
	if err := json.Unmarshal(resp.Result, &rules); err != nil {
		return nil, errors.New(waf.CodeUnavailable, "invalid cloudflare rules payload", err)
	}
	return rules, nil
}
