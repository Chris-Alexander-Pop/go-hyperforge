// Package memory provides an in-memory implementation of ip.IPIntelligence.
package memory

import (
	"context"
	"net"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/ip"
)

// Service implements an in-memory IP intelligence service for testing.
type Service struct {
	mu         *concurrency.SmartRWMutex
	locations  map[string]*ip.GeoLocation
	threats    map[string]*ip.ThreatInfo
	blockedIPs map[string]bool
}

// New creates a new in-memory IP intelligence service.
func New() *Service {
	s := &Service{
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-ip"}),
		locations:  make(map[string]*ip.GeoLocation),
		threats:    make(map[string]*ip.ThreatInfo),
		blockedIPs: make(map[string]bool),
	}

	// Seed with some test data
	s.AddLocation("8.8.8.8", &ip.GeoLocation{
		IP: net.ParseIP("8.8.8.8"), Country: "US", CountryName: "United States",
		Region: "CA", RegionName: "California", City: "Mountain View",
		Latitude: 37.386, Longitude: -122.0838, Timezone: "America/Los_Angeles",
		ASN: 15169, ASNOrg: "Google LLC", ISP: "Google",
	})
	s.AddLocation("1.1.1.1", &ip.GeoLocation{
		IP: net.ParseIP("1.1.1.1"), Country: "AU", CountryName: "Australia",
		City: "Sydney", Latitude: -33.8688, Longitude: 151.2093,
		ASN: 13335, ASNOrg: "Cloudflare, Inc.", ISP: "Cloudflare",
	})

	return s
}

// AddLocation adds a location entry for testing.
func (s *Service) AddLocation(ipAddr string, loc *ip.GeoLocation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locations[ipAddr] = loc
}

// AddThreat adds a threat entry for testing.
func (s *Service) AddThreat(ipAddr string, threat *ip.ThreatInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threats[ipAddr] = threat
}

// BlockIP blocks an IP for testing.
func (s *Service) BlockIP(ipAddr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockedIPs[ipAddr] = true
}

func (s *Service) Lookup(ctx context.Context, ipAddr string) (*ip.GeoLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if loc, ok := s.locations[ipAddr]; ok {
		return loc, nil
	}

	// Return default for unknown IPs
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP == nil {
		return nil, ip.ErrInvalidIP
	}

	return &ip.GeoLocation{
		IP:          parsedIP,
		Country:     "XX",
		CountryName: "Unknown",
	}, nil
}

func (s *Service) LookupBatch(ctx context.Context, ips []string) ([]*ip.GeoLocation, error) {
	results := make([]*ip.GeoLocation, len(ips))
	for i, ipAddr := range ips {
		loc, err := s.Lookup(ctx, ipAddr)
		if err != nil {
			return nil, err
		}
		results[i] = loc
	}
	return results, nil
}

func (s *Service) GetThreatInfo(ctx context.Context, ipAddr string) (*ip.ThreatInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if threat, ok := s.threats[ipAddr]; ok {
		return threat, nil
	}

	parsedIP := net.ParseIP(ipAddr)
	if parsedIP == nil {
		return nil, ip.ErrInvalidIP
	}

	// Return safe default
	return &ip.ThreatInfo{
		IP:          parsedIP,
		IsThreat:    false,
		ThreatLevel: 0,
	}, nil
}

func (s *Service) IsBlocked(ctx context.Context, ipAddr string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.blockedIPs[ipAddr], nil
}

func (s *Service) IsCountryAllowed(ctx context.Context, ipAddr string, allowedCountries []string) (bool, error) {
	loc, err := s.Lookup(ctx, ipAddr)
	if err != nil {
		return false, err
	}

	for _, country := range allowedCountries {
		if loc.Country == country {
			return true, nil
		}
	}
	return false, nil
}
