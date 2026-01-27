// Package dns provides a unified interface for DNS management.
//
// Supported backends:
//   - Memory: In-memory DNS for testing
//   - Route53: AWS Route 53
//   - CloudDNS: Google Cloud DNS
//   - AzureDNS: Azure DNS
//   - Cloudflare: Cloudflare DNS
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/network/dns/adapters/memory"
//
//	manager := memory.New()
//	err := manager.CreateRecord(ctx, dns.Record{Name: "api.example.com", Type: dns.TypeA, Value: "10.0.0.1"})
package dns

import (
	"context"
	"time"
)

// Driver constants for DNS backends.
const (
	DriverMemory     = "memory"
	DriverRoute53    = "route53"
	DriverCloudDNS   = "cloud-dns"
	DriverAzureDNS   = "azure-dns"
	DriverCloudflare = "cloudflare"
)

// RecordType represents DNS record types.
type RecordType string

const (
	TypeA     RecordType = "A"
	TypeAAAA  RecordType = "AAAA"
	TypeCNAME RecordType = "CNAME"
	TypeMX    RecordType = "MX"
	TypeTXT   RecordType = "TXT"
	TypeNS    RecordType = "NS"
	TypeSOA   RecordType = "SOA"
	TypeSRV   RecordType = "SRV"
	TypeCAA   RecordType = "CAA"
	TypePTR   RecordType = "PTR"
)

// Config holds configuration for DNS management.
type Config struct {
	// Driver specifies the DNS backend.
	Driver string `env:"DNS_DRIVER" env-default:"memory"`

	// Zone is the default DNS zone.
	Zone string `env:"DNS_ZONE"`

	// AWS Route53 specific
	AWSAccessKeyID      string `env:"DNS_AWS_ACCESS_KEY"`
	AWSSecretAccessKey  string `env:"DNS_AWS_SECRET_KEY"`
	AWSRegion           string `env:"DNS_AWS_REGION" env-default:"us-east-1"`
	Route53HostedZoneID string `env:"DNS_ROUTE53_ZONE_ID"`

	// GCP Cloud DNS specific
	GCPProjectID string `env:"DNS_GCP_PROJECT"`

	// Azure DNS specific
	AzureSubscriptionID string `env:"DNS_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"DNS_AZURE_RESOURCE_GROUP"`

	// Cloudflare specific
	CloudflareAPIKey   string `env:"DNS_CLOUDFLARE_API_KEY"`
	CloudflareAPIToken string `env:"DNS_CLOUDFLARE_API_TOKEN"`
	CloudflareZoneID   string `env:"DNS_CLOUDFLARE_ZONE_ID"`

	// Common options
	DefaultTTL time.Duration `env:"DNS_DEFAULT_TTL" env-default:"300s"`
	Timeout    time.Duration `env:"DNS_TIMEOUT" env-default:"30s"`
}

// Zone represents a DNS zone.
type Zone struct {
	// ID is the unique identifier for the zone.
	ID string

	// Name is the domain name (e.g., "example.com").
	Name string

	// Comment is a description of the zone.
	Comment string

	// NameServers are the authoritative name servers.
	NameServers []string

	// CreatedAt is when the zone was created.
	CreatedAt time.Time
}

// Record represents a DNS record.
type Record struct {
	// ID is the unique identifier for the record.
	ID string

	// ZoneID is the zone this record belongs to.
	ZoneID string

	// Name is the record name (e.g., "api.example.com").
	Name string

	// Type is the record type (A, AAAA, CNAME, etc.).
	Type RecordType

	// Value is the record value.
	Value string

	// Values allows multiple values for the same record.
	Values []string

	// TTL is the time-to-live in seconds.
	TTL int

	// Priority is used for MX and SRV records.
	Priority int

	// Weight is used for SRV records.
	Weight int

	// Port is used for SRV records.
	Port int

	// Proxied indicates if the record is proxied (Cloudflare).
	Proxied bool

	// CreatedAt is when the record was created.
	CreatedAt time.Time

	// UpdatedAt is when the record was last modified.
	UpdatedAt time.Time
}

// CreateZoneOptions configures zone creation.
type CreateZoneOptions struct {
	// Name is the domain name.
	Name string

	// Comment is a description.
	Comment string
}

// CreateRecordOptions configures record creation.
type CreateRecordOptions struct {
	// ZoneID is the target zone.
	ZoneID string

	// Name is the record name.
	Name string

	// Type is the record type.
	Type RecordType

	// Value is the record value.
	Value string

	// Values allows multiple values.
	Values []string

	// TTL is the time-to-live.
	TTL int

	// Priority for MX/SRV records.
	Priority int

	// Weight for SRV records.
	Weight int

	// Port for SRV records.
	Port int

	// Proxied for Cloudflare.
	Proxied bool
}

// UpdateRecordOptions configures record updates.
type UpdateRecordOptions struct {
	// Value is the new record value.
	Value string

	// Values allows multiple values.
	Values []string

	// TTL is the new TTL.
	TTL int

	// Priority for MX/SRV records.
	Priority int

	// Proxied for Cloudflare.
	Proxied bool
}

// ListRecordsOptions configures record listing.
type ListRecordsOptions struct {
	// Type filters by record type.
	Type RecordType

	// Name filters by record name.
	Name string

	// Limit is the maximum records to return.
	Limit int

	// PageToken is for pagination.
	PageToken string
}

// ListRecordsResult contains the list result.
type ListRecordsResult struct {
	// Records is the list of records.
	Records []*Record

	// NextPageToken is the pagination token.
	NextPageToken string
}

// DNSManager defines the interface for DNS management.
type DNSManager interface {
	// CreateZone creates a new DNS zone.
	CreateZone(ctx context.Context, opts CreateZoneOptions) (*Zone, error)

	// GetZone retrieves a zone by ID.
	GetZone(ctx context.Context, zoneID string) (*Zone, error)

	// ListZones returns all zones.
	ListZones(ctx context.Context) ([]*Zone, error)

	// DeleteZone deletes a zone.
	DeleteZone(ctx context.Context, zoneID string) error

	// CreateRecord creates a new DNS record.
	CreateRecord(ctx context.Context, opts CreateRecordOptions) (*Record, error)

	// GetRecord retrieves a record by ID.
	GetRecord(ctx context.Context, zoneID, recordID string) (*Record, error)

	// ListRecords returns records in a zone.
	ListRecords(ctx context.Context, zoneID string, opts ListRecordsOptions) (*ListRecordsResult, error)

	// UpdateRecord updates an existing record.
	UpdateRecord(ctx context.Context, zoneID, recordID string, opts UpdateRecordOptions) (*Record, error)

	// DeleteRecord deletes a record.
	DeleteRecord(ctx context.Context, zoneID, recordID string) error

	// LookupRecord performs a DNS lookup.
	LookupRecord(ctx context.Context, name string, recordType RecordType) ([]*Record, error)
}
