// Package cdn provides a unified interface for CDN management.
//
// Only the memory adapter ships today. CloudFront / Cloud CDN / Azure CDN /
// Akamai driver constants are reserved placeholders — no cloud adapters yet.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/network/cdn/adapters/memory"
//
//	manager := memory.New()
//	dist, err := manager.CreateDistribution(ctx, cdn.CreateDistributionOptions{OriginDomain: "example.com"})
package cdn

import (
	"context"
	"time"
)

// Driver constants for CDN backends.
// Only DriverMemory ships; cloud drivers are reserved placeholders.
const (
	DriverMemory     = "memory"
	DriverCloudFront = "cloudfront" // reserved — not implemented
	DriverCloudCDN   = "cloudcdn"   // reserved — not implemented
	DriverAzureCDN   = "azurecdn"   // reserved — not implemented
	DriverAkamai     = "akamai"     // reserved — not implemented
)

// DistributionStatus represents the distribution status.
type DistributionStatus string

const (
	StatusDeployed   DistributionStatus = "deployed"
	StatusInProgress DistributionStatus = "in_progress"
	StatusDisabled   DistributionStatus = "disabled"
)

// Config holds configuration for CDN management.
type Config struct {
	// Driver specifies the CDN backend.
	Driver string `env:"CDN_DRIVER" env-default:"memory"`

	// AWS CloudFront specific
	AWSAccessKeyID     string `env:"CDN_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"CDN_AWS_SECRET_KEY"`
	AWSRegion          string `env:"CDN_AWS_REGION" env-default:"us-east-1"`

	// GCP specific
	GCPProjectID string `env:"CDN_GCP_PROJECT"`

	// Azure specific
	AzureSubscriptionID string `env:"CDN_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"CDN_AZURE_RESOURCE_GROUP"`
}

// Distribution represents a CDN distribution.
type Distribution struct {
	// ID is the unique identifier.
	ID string

	// DomainName is the CDN domain.
	DomainName string

	// Status is the distribution status.
	Status DistributionStatus

	// Origins are the origin servers.
	Origins []Origin

	// CacheBehaviors define caching rules.
	CacheBehaviors []CacheBehavior

	// Enabled indicates if distribution is active.
	Enabled bool

	// SSLCertificateARN is the SSL certificate.
	SSLCertificateARN string

	// PriceClass is the pricing tier.
	PriceClass string

	// CreatedAt is when the distribution was created.
	CreatedAt time.Time

	// LastModified is when the distribution was last modified.
	LastModified time.Time
}

// Origin represents an origin server.
type Origin struct {
	// ID is the origin identifier.
	ID string

	// DomainName is the origin domain.
	DomainName string

	// OriginPath is the path prefix.
	OriginPath string

	// Protocol is http, https, or match-viewer.
	Protocol string

	// HTTPPort is the HTTP port.
	HTTPPort int

	// HTTPSPort is the HTTPS port.
	HTTPSPort int
}

// CacheBehavior defines caching rules.
type CacheBehavior struct {
	// PathPattern is the URL pattern.
	PathPattern string

	// OriginID is the target origin.
	OriginID string

	// TTL is the cache TTL in seconds.
	TTL int

	// AllowedMethods are allowed HTTP methods.
	AllowedMethods []string

	// Compress enables gzip compression.
	Compress bool

	// ViewerProtocolPolicy is allow-all, https-only, or redirect-to-https.
	ViewerProtocolPolicy string
}

// CreateDistributionOptions configures distribution creation.
type CreateDistributionOptions struct {
	// OriginDomain is the origin server domain.
	OriginDomain string

	// Aliases are alternate domain names (CNAMEs).
	Aliases []string

	// SSLCertificateARN is the SSL certificate.
	SSLCertificateARN string

	// DefaultTTL is the default cache TTL.
	DefaultTTL int

	// PriceClass is the pricing tier.
	PriceClass string

	// Enabled enables the distribution.
	Enabled bool

	// Tags are key-value metadata.
	Tags map[string]string
}

// CDNManager defines the interface for CDN management.
type CDNManager interface {
	// CreateDistribution creates a new distribution.
	CreateDistribution(ctx context.Context, opts CreateDistributionOptions) (*Distribution, error)

	// GetDistribution retrieves a distribution by ID.
	GetDistribution(ctx context.Context, id string) (*Distribution, error)

	// ListDistributions returns all distributions.
	ListDistributions(ctx context.Context) ([]*Distribution, error)

	// UpdateDistribution updates a distribution.
	UpdateDistribution(ctx context.Context, id string, opts CreateDistributionOptions) (*Distribution, error)

	// DeleteDistribution deletes a distribution.
	DeleteDistribution(ctx context.Context, id string) error

	// DisableDistribution disables a distribution.
	DisableDistribution(ctx context.Context, id string) error

	// EnableDistribution enables a distribution.
	EnableDistribution(ctx context.Context, id string) error

	// Invalidate creates a cache invalidation.
	Invalidate(ctx context.Context, distributionID string, paths []string) (*Invalidation, error)

	// GetInvalidation retrieves an invalidation.
	GetInvalidation(ctx context.Context, distributionID, invalidationID string) (*Invalidation, error)
}

// Invalidation represents a cache invalidation request.
type Invalidation struct {
	// ID is the invalidation ID.
	ID string

	// DistributionID is the target distribution.
	DistributionID string

	// Paths are the paths to invalidate.
	Paths []string

	// Status is the invalidation status.
	Status string

	// CreatedAt is when the invalidation was created.
	CreatedAt time.Time

	// CompletedAt is when the invalidation completed.
	CompletedAt time.Time
}
