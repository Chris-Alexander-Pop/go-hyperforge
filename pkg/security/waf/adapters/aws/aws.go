package aws

import (
	"context"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

// IPSetAPI is the subset of the WAFv2 client used by this adapter.
// *wafv2.Client satisfies this interface.
type IPSetAPI interface {
	GetIPSet(ctx context.Context, params *wafv2.GetIPSetInput, optFns ...func(*wafv2.Options)) (*wafv2.GetIPSetOutput, error)
	UpdateIPSet(ctx context.Context, params *wafv2.UpdateIPSetInput, optFns ...func(*wafv2.Options)) (*wafv2.UpdateIPSetOutput, error)
}

// Config configures the AWS WAF IP set adapter.
type Config struct {
	// Region is the AWS region (required for New; ignored for CLOUDFRONT scope in some setups).
	Region string `env:"AWS_REGION" env-default:"us-east-1"`

	// AccessKeyID / SecretAccessKey are optional static credentials.
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`

	// IPSetID is the WAFv2 IP set identifier.
	IPSetID string `env:"AWS_WAF_IPSET_ID" validate:"required"`

	// IPSetName is the WAFv2 IP set name.
	IPSetName string `env:"AWS_WAF_IPSET_NAME" validate:"required"`

	// Scope is REGIONAL or CLOUDFRONT (default REGIONAL).
	Scope string `env:"AWS_WAF_SCOPE" env-default:"REGIONAL"`

	// Description is written on UpdateIPSet when non-empty.
	Description string `env:"AWS_WAF_IPSET_DESCRIPTION"`
}

// Validate checks required Config fields.
func (c Config) Validate() error {
	if err := validator.New().ValidateStruct(context.Background(), c); err != nil {
		if errors.IsCode(err, errors.CodeInvalidArgument) {
			return err
		}
		return errors.New(waf.CodeInvalidRule, "invalid aws waf config", err)
	}
	return nil
}

// Manager implements waf.Manager via AWS WAFv2 IP set updates.
type Manager struct {
	client      IPSetAPI
	ipSetID     string
	ipSetName   string
	scope       types.Scope
	description string
}

// Ensure Manager implements waf.Manager.
var _ waf.Manager = (*Manager)(nil)

// NewFromAPI wraps an existing IPSetAPI (SDK client or test double).
func NewFromAPI(api IPSetAPI, cfg Config) (*Manager, error) {
	if api == nil {
		return nil, errors.New(waf.CodeInvalidRule, "waf api client is required", nil)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	var scope types.Scope
	switch strings.ToUpper(cfg.Scope) {
	case "CLOUDFRONT":
		scope = types.ScopeCloudfront
	case "", "REGIONAL":
		scope = types.ScopeRegional
	default:
		return nil, errors.New(waf.CodeInvalidRule, "aws waf scope must be REGIONAL or CLOUDFRONT", nil)
	}
	return &Manager{
		client:      api,
		ipSetID:     cfg.IPSetID,
		ipSetName:   cfg.IPSetName,
		scope:       scope,
		description: cfg.Description,
	}, nil
}

// New builds a Manager from AWS SDK config.
func New(ctx context.Context, cfg Config) (*Manager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, errors.New(waf.CodeInvalidRule, "aws region is required", nil)
	}
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.New(waf.CodeUnavailable, "failed to load aws config", err)
	}
	return NewFromAPI(wafv2.NewFromConfig(awsCfg), cfg)
}

func normalizeCIDR(ip string) (string, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", waf.ErrInvalidRule
	}
	if strings.Contains(ip, "/") {
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			return "", waf.ErrInvalidRule
		}
		return ip, nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", waf.ErrInvalidRule
	}
	if parsed.To4() != nil {
		return ip + "/32", nil
	}
	return ip + "/128", nil
}

func (m *Manager) getIPSet(ctx context.Context) (*types.IPSet, string, error) {
	out, err := m.client.GetIPSet(ctx, &wafv2.GetIPSetInput{
		Id:    aws.String(m.ipSetID),
		Name:  aws.String(m.ipSetName),
		Scope: m.scope,
	})
	if err != nil {
		return nil, "", errors.New(waf.CodeUnavailable, "aws waf get ipset failed", err)
	}
	if out == nil || out.IPSet == nil || out.LockToken == nil {
		return nil, "", errors.New(waf.CodeUnavailable, "aws waf get ipset returned empty", nil)
	}
	return out.IPSet, *out.LockToken, nil
}

func (m *Manager) updateAddresses(ctx context.Context, addresses []string, lockToken string) error {
	in := &wafv2.UpdateIPSetInput{
		Id:        aws.String(m.ipSetID),
		Name:      aws.String(m.ipSetName),
		Scope:     m.scope,
		Addresses: addresses,
		LockToken: aws.String(lockToken),
	}
	if m.description != "" {
		in.Description = aws.String(m.description)
	}
	_, err := m.client.UpdateIPSet(ctx, in)
	if err != nil {
		return errors.New(waf.CodeUnavailable, "aws waf update ipset failed", err)
	}
	return nil
}

// BlockIP adds the IP (as /32 or /128 CIDR) to the IP set.
func (m *Manager) BlockIP(ctx context.Context, ip, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cidr, err := normalizeCIDR(ip)
	if err != nil {
		return err
	}
	ipset, lock, err := m.getIPSet(ctx)
	if err != nil {
		return err
	}
	for _, a := range ipset.Addresses {
		if a == cidr {
			return nil // already blocked
		}
	}
	addrs := append(append([]string{}, ipset.Addresses...), cidr)
	_ = reason // WAFv2 IP sets have no per-address notes; Description is set-level.
	return m.updateAddresses(ctx, addrs, lock)
}

// AllowIP removes the IP from the IP set.
func (m *Manager) AllowIP(ctx context.Context, ip string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cidr, err := normalizeCIDR(ip)
	if err != nil {
		return err
	}
	ipset, lock, err := m.getIPSet(ctx)
	if err != nil {
		return err
	}
	found := false
	next := make([]string, 0, len(ipset.Addresses))
	for _, a := range ipset.Addresses {
		if a == cidr {
			found = true
			continue
		}
		next = append(next, a)
	}
	if !found {
		return waf.ErrNotFound
	}
	return m.updateAddresses(ctx, next, lock)
}

// GetRules lists addresses in the IP set as block rules.
func (m *Manager) GetRules(ctx context.Context) ([]waf.Rule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ipset, _, err := m.getIPSet(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]waf.Rule, 0, len(ipset.Addresses))
	for i, a := range ipset.Addresses {
		rule := waf.Rule{
			ID:     a,
			Action: "block",
			Reason: "aws-waf-ipset",
		}
		if strings.HasSuffix(a, "/32") || strings.HasSuffix(a, "/128") {
			rule.IP = strings.Split(a, "/")[0]
		} else {
			rule.CIDR = a
		}
		if rule.ID == "" {
			rule.ID = string(rune('a' + i))
		}
		out = append(out, rule)
	}
	return out, nil
}
