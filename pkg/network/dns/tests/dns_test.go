package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/network/dns"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/dns/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// DNSManagerSuite provides a generic test suite for DNSManager implementations.
type DNSManagerSuite struct {
	suite.Suite
	manager dns.DNSManager
	ctx     context.Context
}

// SetupTest runs before each test.
func (s *DNSManagerSuite) SetupTest() {
	s.manager = memory.New()
	s.ctx = context.Background()
}

func (s *DNSManagerSuite) TestCreateAndGetZone() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{
		Name:    "example.com",
		Comment: "Test zone",
	})
	s.Require().NoError(err)
	s.NotEmpty(zone.ID)
	s.Equal("example.com", zone.Name)
	s.NotEmpty(zone.NameServers)

	got, err := s.manager.GetZone(s.ctx, zone.ID)
	s.Require().NoError(err)
	s.Equal(zone.ID, got.ID)
	s.Equal(zone.Name, got.Name)
}

func (s *DNSManagerSuite) TestCreateZoneAlreadyExists() {
	_, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "duplicate.com"})
	s.Require().NoError(err)

	_, err = s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "duplicate.com"})
	s.Error(err)
}

func (s *DNSManagerSuite) TestListZones() {
	for i := 0; i < 3; i++ {
		_, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{
			Name: "zone" + string(rune('0'+i)) + ".com",
		})
		s.Require().NoError(err)
	}

	zones, err := s.manager.ListZones(s.ctx)
	s.Require().NoError(err)
	s.Len(zones, 3)
}

func (s *DNSManagerSuite) TestDeleteZone() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "delete-me.com"})
	s.Require().NoError(err)

	err = s.manager.DeleteZone(s.ctx, zone.ID)
	s.Require().NoError(err)

	_, err = s.manager.GetZone(s.ctx, zone.ID)
	s.Error(err)
}

func (s *DNSManagerSuite) TestDeleteZoneNotFound() {
	err := s.manager.DeleteZone(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *DNSManagerSuite) TestCreateAndGetRecord() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	record, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID,
		Name:   "api.example.com",
		Type:   dns.TypeA,
		Value:  "10.0.0.1",
		TTL:    300,
	})
	s.Require().NoError(err)
	s.NotEmpty(record.ID)
	s.Equal("api.example.com", record.Name)
	s.Equal(dns.TypeA, record.Type)
	s.Equal("10.0.0.1", record.Value)

	got, err := s.manager.GetRecord(s.ctx, zone.ID, record.ID)
	s.Require().NoError(err)
	s.Equal(record.ID, got.ID)
}

func (s *DNSManagerSuite) TestCreateRecordZoneNotFound() {
	_, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: "nonexistent",
		Name:   "test.example.com",
		Type:   dns.TypeA,
		Value:  "10.0.0.1",
	})
	s.Error(err)
}

func (s *DNSManagerSuite) TestListRecords() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	// Create some records
	for i := 0; i < 5; i++ {
		_, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
			ZoneID: zone.ID,
			Name:   "host" + string(rune('0'+i)) + ".example.com",
			Type:   dns.TypeA,
			Value:  "10.0.0." + string(rune('1'+i)),
		})
		s.Require().NoError(err)
	}

	result, err := s.manager.ListRecords(s.ctx, zone.ID, dns.ListRecordsOptions{})
	s.Require().NoError(err)
	s.Len(result.Records, 5)
}

func (s *DNSManagerSuite) TestListRecordsWithTypeFilter() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	// Create mixed records
	_, err = s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID, Name: "a.example.com", Type: dns.TypeA, Value: "10.0.0.1",
	})
	s.Require().NoError(err)

	_, err = s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID, Name: "cname.example.com", Type: dns.TypeCNAME, Value: "target.example.com",
	})
	s.Require().NoError(err)

	// Filter by type
	result, err := s.manager.ListRecords(s.ctx, zone.ID, dns.ListRecordsOptions{Type: dns.TypeA})
	s.Require().NoError(err)
	s.Len(result.Records, 1)
	s.Equal(dns.TypeA, result.Records[0].Type)
}

func (s *DNSManagerSuite) TestUpdateRecord() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	record, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID,
		Name:   "api.example.com",
		Type:   dns.TypeA,
		Value:  "10.0.0.1",
		TTL:    300,
	})
	s.Require().NoError(err)

	updated, err := s.manager.UpdateRecord(s.ctx, zone.ID, record.ID, dns.UpdateRecordOptions{
		Value: "10.0.0.2",
		TTL:   600,
	})
	s.Require().NoError(err)
	s.Equal("10.0.0.2", updated.Value)
	s.Equal(600, updated.TTL)
}

func (s *DNSManagerSuite) TestDeleteRecord() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	record, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID,
		Name:   "delete-me.example.com",
		Type:   dns.TypeA,
		Value:  "10.0.0.1",
	})
	s.Require().NoError(err)

	err = s.manager.DeleteRecord(s.ctx, zone.ID, record.ID)
	s.Require().NoError(err)

	_, err = s.manager.GetRecord(s.ctx, zone.ID, record.ID)
	s.Error(err)
}

func (s *DNSManagerSuite) TestLookupRecord() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	_, err = s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID,
		Name:   "api.example.com",
		Type:   dns.TypeA,
		Value:  "10.0.0.1",
	})
	s.Require().NoError(err)

	records, err := s.manager.LookupRecord(s.ctx, "api.example.com", dns.TypeA)
	s.Require().NoError(err)
	s.NotEmpty(records)
	s.Equal("10.0.0.1", records[0].Value)
}

func (s *DNSManagerSuite) TestCNAMERecord() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	record, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID: zone.ID,
		Name:   "www.example.com",
		Type:   dns.TypeCNAME,
		Value:  "example.com",
		TTL:    3600,
	})
	s.Require().NoError(err)
	s.Equal(dns.TypeCNAME, record.Type)
	s.Equal("example.com", record.Value)
}

func (s *DNSManagerSuite) TestMXRecord() {
	zone, err := s.manager.CreateZone(s.ctx, dns.CreateZoneOptions{Name: "example.com"})
	s.Require().NoError(err)

	record, err := s.manager.CreateRecord(s.ctx, dns.CreateRecordOptions{
		ZoneID:   zone.ID,
		Name:     "example.com",
		Type:     dns.TypeMX,
		Value:    "mail.example.com",
		Priority: 10,
		TTL:      3600,
	})
	s.Require().NoError(err)
	s.Equal(dns.TypeMX, record.Type)
	s.Equal(10, record.Priority)
}

// TestDNSManagerSuite runs the test suite.
func TestDNSManagerSuite(t *testing.T) {
	suite.Run(t, new(DNSManagerSuite))
}
