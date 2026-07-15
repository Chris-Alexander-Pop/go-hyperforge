package waf

import (
	"context"
)

// Config configures the WAF.
type Config struct {
	// Provider selects the WAF backend.
	// Implemented: "memory", "cloudflare". Reserved: "aws-waf".
	Provider string `env:"SECURITY_WAF_PROVIDER" env-default:"memory"`
}

// Rule represents a blocking rule.
type Rule struct {
	ID        string `json:"id"`
	IP        string `json:"ip,omitempty"`
	CIDR      string `json:"cidr,omitempty"`
	Action    string `json:"action"` // block, allow
	Reason    string `json:"reason"`
	ExpiresAt int64  `json:"expires_at,omitempty"` // Unix timestamp
}

// Manager defines the interface for WAF operations.
type Manager interface {
	BlockIP(ctx context.Context, ip, reason string) error
	AllowIP(ctx context.Context, ip string) error
	GetRules(ctx context.Context) ([]Rule, error)
}
