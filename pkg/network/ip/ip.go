// Package ip provides IP intelligence and geolocation services.
//
// Features (memory adapter today):
//   - IP geolocation (country, city, region)
//   - IP reputation and threat detection
//   - ASN and ISP lookup
//   - VPN/Proxy/Tor detection
//
// Only the memory adapter ships today. MaxMind / IPInfo / IPStack driver
// constants are reserved placeholders — no external providers yet.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/network/ip/adapters/memory"
//
//	geo := memory.New()
//	loc, err := geo.Lookup(ctx, "8.8.8.8")
package ip

import (
	"context"
	"net"
)

// Driver constants for IP intelligence backends.
// Only DriverMemory ships; provider drivers are reserved placeholders.
const (
	DriverMemory  = "memory"
	DriverMaxMind = "maxmind" // reserved — not implemented
	DriverIPInfo  = "ipinfo"  // reserved — not implemented
	DriverIPStack = "ipstack" // reserved — not implemented
)

// Config holds configuration for IP intelligence.
type Config struct {
	// Driver specifies the IP intelligence backend.
	Driver string `env:"IP_DRIVER" env-default:"memory"`

	// MaxMind specific
	MaxMindAccountID  string `env:"MAXMIND_ACCOUNT_ID"`
	MaxMindLicenseKey string `env:"MAXMIND_LICENSE_KEY"`
	MaxMindDBPath     string `env:"MAXMIND_DB_PATH"`

	// IPInfo specific
	IPInfoToken string `env:"IPINFO_TOKEN"`

	// IPStack specific
	IPStackKey string `env:"IPSTACK_KEY"`
}

// GeoLocation represents IP geolocation data.
type GeoLocation struct {
	// IP is the queried IP address.
	IP net.IP

	// Country is the ISO country code.
	Country string

	// CountryName is the full country name.
	CountryName string

	// Region is the region/state code.
	Region string

	// RegionName is the full region/state name.
	RegionName string

	// City is the city name.
	City string

	// PostalCode is the postal/zip code.
	PostalCode string

	// Latitude is the latitude coordinate.
	Latitude float64

	// Longitude is the longitude coordinate.
	Longitude float64

	// Timezone is the timezone name.
	Timezone string

	// ASN is the autonomous system number.
	ASN int

	// ASNOrg is the ASN organization name.
	ASNOrg string

	// ISP is the Internet service provider.
	ISP string
}

// ThreatInfo represents IP threat/reputation data.
type ThreatInfo struct {
	// IP is the queried IP address.
	IP net.IP

	// IsThreat indicates if IP is known malicious.
	IsThreat bool

	// ThreatLevel is the threat level (0-100).
	ThreatLevel int

	// Categories are threat categories.
	Categories []string

	// IsVPN indicates if IP is a VPN exit.
	IsVPN bool

	// IsProxy indicates if IP is a proxy.
	IsProxy bool

	// IsTor indicates if IP is a Tor exit node.
	IsTor bool

	// IsBot indicates if IP is known bot traffic.
	IsBot bool

	// IsDatacenter indicates if IP is from a datacenter.
	IsDatacenter bool
}

// IPIntelligence defines the interface for IP intelligence services.
type IPIntelligence interface {
	// Lookup returns geolocation for an IP address.
	Lookup(ctx context.Context, ip string) (*GeoLocation, error)

	// LookupBatch returns geolocation for multiple IPs.
	LookupBatch(ctx context.Context, ips []string) ([]*GeoLocation, error)

	// GetThreatInfo returns threat/reputation info for an IP.
	GetThreatInfo(ctx context.Context, ip string) (*ThreatInfo, error)

	// IsBlocked checks if an IP should be blocked.
	IsBlocked(ctx context.Context, ip string) (bool, error)

	// IsCountryAllowed checks if an IP's country is in the allowed list.
	IsCountryAllowed(ctx context.Context, ip string, allowedCountries []string) (bool, error)
}
