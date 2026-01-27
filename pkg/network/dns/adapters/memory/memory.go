package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/dns"
	"github.com/google/uuid"
)

// Manager implements an in-memory DNS manager for testing.
type Manager struct {
	mu      sync.RWMutex
	zones   map[string]*dns.Zone
	records map[string]map[string]*dns.Record // zoneID -> recordID -> record
	config  dns.Config
}

// New creates a new in-memory DNS manager.
func New() *Manager {
	return &Manager{
		zones:   make(map[string]*dns.Zone),
		records: make(map[string]map[string]*dns.Record),
		config:  dns.Config{DefaultTTL: 300 * time.Second},
	}
}

// NewWithConfig creates a new in-memory DNS manager with config.
func NewWithConfig(cfg dns.Config) *Manager {
	m := New()
	m.config = cfg
	return m
}

func (m *Manager) CreateZone(ctx context.Context, opts dns.CreateZoneOptions) (*dns.Zone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if zone already exists
	for _, zone := range m.zones {
		if zone.Name == opts.Name {
			return nil, errors.Conflict("zone already exists", nil)
		}
	}

	zone := &dns.Zone{
		ID:      uuid.NewString(),
		Name:    opts.Name,
		Comment: opts.Comment,
		NameServers: []string{
			"ns1.example.com",
			"ns2.example.com",
		},
		CreatedAt: time.Now(),
	}

	m.zones[zone.ID] = zone
	m.records[zone.ID] = make(map[string]*dns.Record)

	return zone, nil
}

func (m *Manager) GetZone(ctx context.Context, zoneID string) (*dns.Zone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zone, ok := m.zones[zoneID]
	if !ok {
		return nil, errors.NotFound("zone not found", nil)
	}

	return zone, nil
}

func (m *Manager) ListZones(ctx context.Context) ([]*dns.Zone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zones := make([]*dns.Zone, 0, len(m.zones))
	for _, zone := range m.zones {
		zones = append(zones, zone)
	}

	return zones, nil
}

func (m *Manager) DeleteZone(ctx context.Context, zoneID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.zones[zoneID]; !ok {
		return errors.NotFound("zone not found", nil)
	}

	delete(m.zones, zoneID)
	delete(m.records, zoneID)

	return nil
}

func (m *Manager) CreateRecord(ctx context.Context, opts dns.CreateRecordOptions) (*dns.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.zones[opts.ZoneID]; !ok {
		return nil, errors.NotFound("zone not found", nil)
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = int(m.config.DefaultTTL.Seconds())
	}

	now := time.Now()
	record := &dns.Record{
		ID:        uuid.NewString(),
		ZoneID:    opts.ZoneID,
		Name:      opts.Name,
		Type:      opts.Type,
		Value:     opts.Value,
		Values:    opts.Values,
		TTL:       ttl,
		Priority:  opts.Priority,
		Weight:    opts.Weight,
		Port:      opts.Port,
		Proxied:   opts.Proxied,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if record.Values == nil && record.Value != "" {
		record.Values = []string{record.Value}
	}

	m.records[opts.ZoneID][record.ID] = record

	return record, nil
}

func (m *Manager) GetRecord(ctx context.Context, zoneID, recordID string) (*dns.Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zoneRecords, ok := m.records[zoneID]
	if !ok {
		return nil, errors.NotFound("zone not found", nil)
	}

	record, ok := zoneRecords[recordID]
	if !ok {
		return nil, errors.NotFound("record not found", nil)
	}

	return record, nil
}

func (m *Manager) ListRecords(ctx context.Context, zoneID string, opts dns.ListRecordsOptions) (*dns.ListRecordsResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zoneRecords, ok := m.records[zoneID]
	if !ok {
		return nil, errors.NotFound("zone not found", nil)
	}

	result := &dns.ListRecordsResult{
		Records: make([]*dns.Record, 0),
	}

	for _, record := range zoneRecords {
		// Apply filters
		if opts.Type != "" && record.Type != opts.Type {
			continue
		}
		if opts.Name != "" && record.Name != opts.Name {
			continue
		}
		result.Records = append(result.Records, record)
	}

	// Apply limit
	if opts.Limit > 0 && len(result.Records) > opts.Limit {
		result.Records = result.Records[:opts.Limit]
		result.NextPageToken = "more"
	}

	return result, nil
}

func (m *Manager) UpdateRecord(ctx context.Context, zoneID, recordID string, opts dns.UpdateRecordOptions) (*dns.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	zoneRecords, ok := m.records[zoneID]
	if !ok {
		return nil, errors.NotFound("zone not found", nil)
	}

	record, ok := zoneRecords[recordID]
	if !ok {
		return nil, errors.NotFound("record not found", nil)
	}

	if opts.Value != "" {
		record.Value = opts.Value
		record.Values = []string{opts.Value}
	}
	if len(opts.Values) > 0 {
		record.Values = opts.Values
		if len(opts.Values) > 0 {
			record.Value = opts.Values[0]
		}
	}
	if opts.TTL > 0 {
		record.TTL = opts.TTL
	}
	if opts.Priority > 0 {
		record.Priority = opts.Priority
	}
	record.Proxied = opts.Proxied
	record.UpdatedAt = time.Now()

	return record, nil
}

func (m *Manager) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	zoneRecords, ok := m.records[zoneID]
	if !ok {
		return errors.NotFound("zone not found", nil)
	}

	if _, ok := zoneRecords[recordID]; !ok {
		return errors.NotFound("record not found", nil)
	}

	delete(zoneRecords, recordID)

	return nil
}

func (m *Manager) LookupRecord(ctx context.Context, name string, recordType dns.RecordType) ([]*dns.Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*dns.Record

	for _, zoneRecords := range m.records {
		for _, record := range zoneRecords {
			if record.Type != recordType {
				continue
			}
			// Match exact name or subdomain
			if record.Name == name || strings.HasSuffix(name, "."+record.Name) {
				results = append(results, record)
			}
		}
	}

	return results, nil
}
