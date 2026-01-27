package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/discovery"
	"github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/discovery/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// ServiceRegistrySuite provides a generic test suite for ServiceRegistry implementations.
type ServiceRegistrySuite struct {
	suite.Suite
	registry discovery.ServiceRegistry
	ctx      context.Context
}

// SetupTest runs before each test.
func (s *ServiceRegistrySuite) SetupTest() {
	s.registry = memory.New()
	s.ctx = context.Background()
}

func (s *ServiceRegistrySuite) TearDownTest() {
	if s.registry != nil {
		s.registry.Close()
	}
}

func (s *ServiceRegistrySuite) TestRegisterAndGet() {
	svc, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name:    "api-service",
		Address: "10.0.0.1",
		Port:    8080,
		Tags:    []string{"v1", "production"},
	})
	s.Require().NoError(err)
	s.NotEmpty(svc.ID)
	s.Equal("api-service", svc.Name)
	s.Equal("10.0.0.1", svc.Address)
	s.Equal(discovery.HealthStatusPassing, svc.Health)

	got, err := s.registry.Get(s.ctx, svc.ID)
	s.Require().NoError(err)
	s.Equal(svc.ID, got.ID)
}

func (s *ServiceRegistrySuite) TestRegisterMissingName() {
	_, err := s.registry.Register(s.ctx, discovery.RegisterOptions{Address: "10.0.0.1"})
	s.Error(err)
}

func (s *ServiceRegistrySuite) TestDeregister() {
	svc, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name:    "temp-service",
		Address: "10.0.0.1",
		Port:    8080,
	})
	s.Require().NoError(err)

	err = s.registry.Deregister(s.ctx, svc.ID)
	s.Require().NoError(err)

	_, err = s.registry.Get(s.ctx, svc.ID)
	s.Error(err)
}

func (s *ServiceRegistrySuite) TestDeregisterNotFound() {
	err := s.registry.Deregister(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *ServiceRegistrySuite) TestLookup() {
	// Register multiple instances
	for i := 0; i < 3; i++ {
		_, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
			Name:    "api-service",
			Address: "10.0.0." + string(rune('1'+i)),
			Port:    8080,
		})
		s.Require().NoError(err)
	}

	// Register different service
	_, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name:    "other-service",
		Address: "10.0.1.1",
		Port:    9090,
	})
	s.Require().NoError(err)

	services, err := s.registry.Lookup(s.ctx, "api-service", discovery.QueryOptions{})
	s.Require().NoError(err)
	s.Len(services, 3)
}

func (s *ServiceRegistrySuite) TestLookupWithHealthFilter() {
	svc1, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.1", Port: 8080,
	})
	s.Require().NoError(err)

	svc2, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.2", Port: 8080,
	})
	s.Require().NoError(err)

	// Mark one as unhealthy
	err = s.registry.UpdateHealth(s.ctx, svc2.ID, discovery.HealthStatusCritical)
	s.Require().NoError(err)

	// Lookup healthy only
	services, err := s.registry.Lookup(s.ctx, "api", discovery.QueryOptions{HealthyOnly: true})
	s.Require().NoError(err)
	s.Len(services, 1)
	s.Equal(svc1.ID, services[0].ID)
}

func (s *ServiceRegistrySuite) TestLookupWithTagFilter() {
	_, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.1", Port: 8080, Tags: []string{"v1"},
	})
	s.Require().NoError(err)

	_, err = s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.2", Port: 8080, Tags: []string{"v2"},
	})
	s.Require().NoError(err)

	services, err := s.registry.Lookup(s.ctx, "api", discovery.QueryOptions{Tag: "v1"})
	s.Require().NoError(err)
	s.Len(services, 1)
}

func (s *ServiceRegistrySuite) TestList() {
	for i := 0; i < 5; i++ {
		_, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
			Name:    "service-" + string(rune('a'+i)),
			Address: "10.0.0." + string(rune('1'+i)),
			Port:    8080,
		})
		s.Require().NoError(err)
	}

	services, err := s.registry.List(s.ctx, discovery.QueryOptions{})
	s.Require().NoError(err)
	s.Len(services, 5)
}

func (s *ServiceRegistrySuite) TestHeartbeat() {
	svc, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.1", Port: 8080,
	})
	s.Require().NoError(err)

	originalTime := svc.LastHeartbeat
	time.Sleep(10 * time.Millisecond)

	err = s.registry.Heartbeat(s.ctx, svc.ID)
	s.Require().NoError(err)

	svc, err = s.registry.Get(s.ctx, svc.ID)
	s.Require().NoError(err)
	s.True(svc.LastHeartbeat.After(originalTime))
}

func (s *ServiceRegistrySuite) TestUpdateHealth() {
	svc, err := s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.1", Port: 8080,
	})
	s.Require().NoError(err)
	s.Equal(discovery.HealthStatusPassing, svc.Health)

	err = s.registry.UpdateHealth(s.ctx, svc.ID, discovery.HealthStatusWarning)
	s.Require().NoError(err)

	svc, err = s.registry.Get(s.ctx, svc.ID)
	s.Require().NoError(err)
	s.Equal(discovery.HealthStatusWarning, svc.Health)
}

func (s *ServiceRegistrySuite) TestWatch() {
	ch, err := s.registry.Watch(s.ctx, "api")
	s.Require().NoError(err)

	// Register a service
	_, err = s.registry.Register(s.ctx, discovery.RegisterOptions{
		Name: "api", Address: "10.0.0.1", Port: 8080,
	})
	s.Require().NoError(err)

	// Should receive update
	select {
	case services := <-ch:
		s.NotEmpty(services)
	case <-time.After(100 * time.Millisecond):
		s.Fail("timeout waiting for watch update")
	}
}

// TestServiceRegistrySuite runs the test suite.
func TestServiceRegistrySuite(t *testing.T) {
	suite.Run(t, new(ServiceRegistrySuite))
}
