package scanning

import (
	"context"
	"time"
)

// Config configures the Scanner.
type Config struct {
	// Provider specifies the scanning provider.
	// Implemented: "memory", "guardduty" (adapters/guardduty).
	// Reserved: "clamav".
	Provider string `env:"SECURITY_SCANNING_PROVIDER" env-default:"memory"`
}

// Report represents a scan result.
type Report struct {
	ResourceID string    `json:"resource_id"`
	Clean      bool      `json:"clean"`
	Threats    []string  `json:"threats,omitempty"`
	ScannedAt  time.Time `json:"scanned_at"`
}

// Resource represents an item to scan (file path, url, blob pointer).
type Resource struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // file, url, s3-object
	Location string `json:"location"`
}

// Scanner defines the interface for vulnerability/malware scanning.
type Scanner interface {
	Scan(ctx context.Context, resource Resource) (*Report, error)
}
