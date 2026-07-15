package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/network/loadbalancer"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/loadbalancer/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// LoadBalancerManagerSuite provides a generic test suite for LoadBalancerManager implementations.
type LoadBalancerManagerSuite struct {
	suite.Suite
	manager loadbalancer.LoadBalancerManager
	ctx     context.Context
}

// SetupTest runs before each test.
func (s *LoadBalancerManagerSuite) SetupTest() {
	s.manager = memory.New()
	s.ctx = context.Background()
}

func (s *LoadBalancerManagerSuite) TestCreateAndGetLoadBalancer() {
	lb, err := s.manager.CreateLoadBalancer(s.ctx, loadbalancer.CreateLoadBalancerOptions{
		Name:   "test-lb",
		Scheme: "internet-facing",
		Type:   "application",
		Tags:   map[string]string{"env": "test"},
	})
	s.Require().NoError(err)
	s.NotEmpty(lb.ID)
	s.Equal("test-lb", lb.Name)
	s.NotEmpty(lb.DNSName)

	got, err := s.manager.GetLoadBalancer(s.ctx, lb.ID)
	s.Require().NoError(err)
	s.Equal(lb.ID, got.ID)
}

func (s *LoadBalancerManagerSuite) TestListLoadBalancers() {
	for i := 0; i < 3; i++ {
		_, err := s.manager.CreateLoadBalancer(s.ctx, loadbalancer.CreateLoadBalancerOptions{
			Name: "lb-" + string(rune('0'+i)),
		})
		s.Require().NoError(err)
	}

	lbs, err := s.manager.ListLoadBalancers(s.ctx)
	s.Require().NoError(err)
	s.Len(lbs, 3)
}

func (s *LoadBalancerManagerSuite) TestDeleteLoadBalancer() {
	lb, err := s.manager.CreateLoadBalancer(s.ctx, loadbalancer.CreateLoadBalancerOptions{Name: "delete-me"})
	s.Require().NoError(err)

	err = s.manager.DeleteLoadBalancer(s.ctx, lb.ID)
	s.Require().NoError(err)

	_, err = s.manager.GetLoadBalancer(s.ctx, lb.ID)
	s.True(errors.Is(err, loadbalancer.ErrLoadBalancerNotFound))
}

func (s *LoadBalancerManagerSuite) TestDeleteLoadBalancerNotFound() {
	err := s.manager.DeleteLoadBalancer(s.ctx, "nonexistent")
	s.True(errors.Is(err, loadbalancer.ErrLoadBalancerNotFound))
}

func (s *LoadBalancerManagerSuite) TestCreateAndDeleteListener() {
	lb, err := s.manager.CreateLoadBalancer(s.ctx, loadbalancer.CreateLoadBalancerOptions{Name: "listener-test"})
	s.Require().NoError(err)

	pool, err := s.manager.CreateTargetPool(s.ctx, loadbalancer.CreateTargetPoolOptions{
		Name:     "target-pool",
		Protocol: loadbalancer.ProtocolHTTP,
		Port:     8080,
	})
	s.Require().NoError(err)

	listener, err := s.manager.CreateListener(s.ctx, loadbalancer.CreateListenerOptions{
		LoadBalancerID: lb.ID,
		Protocol:       loadbalancer.ProtocolHTTP,
		Port:           80,
		TargetPoolID:   pool.ID,
	})
	s.Require().NoError(err)
	s.NotEmpty(listener.ID)
	s.Equal(80, listener.Port)

	// Verify listener is attached
	lb, err = s.manager.GetLoadBalancer(s.ctx, lb.ID)
	s.Require().NoError(err)
	s.Len(lb.Listeners, 1)

	// Delete listener
	err = s.manager.DeleteListener(s.ctx, lb.ID, listener.ID)
	s.Require().NoError(err)

	lb, err = s.manager.GetLoadBalancer(s.ctx, lb.ID)
	s.Require().NoError(err)
	s.Len(lb.Listeners, 0)
}

func (s *LoadBalancerManagerSuite) TestCreateTargetPool() {
	pool, err := s.manager.CreateTargetPool(s.ctx, loadbalancer.CreateTargetPoolOptions{
		Name:      "api-pool",
		Protocol:  loadbalancer.ProtocolHTTP,
		Port:      8080,
		Algorithm: loadbalancer.AlgorithmLeastConnections,
		HealthCheck: &loadbalancer.HealthCheck{
			Type:               loadbalancer.HealthCheckHTTP,
			Path:               "/health",
			IntervalSeconds:    30,
			TimeoutSeconds:     5,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		},
	})
	s.Require().NoError(err)
	s.NotEmpty(pool.ID)
	s.Equal("api-pool", pool.Name)
	s.Equal(loadbalancer.AlgorithmLeastConnections, pool.Algorithm)
	s.NotNil(pool.HealthCheck)
}

func (s *LoadBalancerManagerSuite) TestAddAndRemoveTarget() {
	pool, err := s.manager.CreateTargetPool(s.ctx, loadbalancer.CreateTargetPoolOptions{
		Name:     "test-pool",
		Protocol: loadbalancer.ProtocolHTTP,
		Port:     8080,
	})
	s.Require().NoError(err)

	err = s.manager.AddTarget(s.ctx, pool.ID, loadbalancer.Target{
		Address: "10.0.0.1",
		Port:    8080,
		Weight:  10,
	})
	s.Require().NoError(err)

	err = s.manager.AddTarget(s.ctx, pool.ID, loadbalancer.Target{
		Address: "10.0.0.2",
		Port:    8080,
	})
	s.Require().NoError(err)

	// Get target health
	targets, err := s.manager.GetTargetHealth(s.ctx, pool.ID)
	s.Require().NoError(err)
	s.Len(targets, 2)

	// Verify first target
	s.Equal("10.0.0.1", targets[0].Address)
	s.Equal(loadbalancer.TargetStatusHealthy, targets[0].Status)

	// Remove target
	err = s.manager.RemoveTarget(s.ctx, pool.ID, targets[0].ID)
	s.Require().NoError(err)

	targets, err = s.manager.GetTargetHealth(s.ctx, pool.ID)
	s.Require().NoError(err)
	s.Len(targets, 1)
}

func (s *LoadBalancerManagerSuite) TestAddDuplicateTarget() {
	pool, err := s.manager.CreateTargetPool(s.ctx, loadbalancer.CreateTargetPoolOptions{
		Name:     "dup-pool",
		Protocol: loadbalancer.ProtocolHTTP,
		Port:     8080,
	})
	s.Require().NoError(err)

	err = s.manager.AddTarget(s.ctx, pool.ID, loadbalancer.Target{Address: "10.0.0.1", Port: 8080})
	s.Require().NoError(err)

	err = s.manager.AddTarget(s.ctx, pool.ID, loadbalancer.Target{Address: "10.0.0.1", Port: 8080})
	s.True(errors.Is(err, loadbalancer.ErrTargetAlreadyRegistered))
}

func (s *LoadBalancerManagerSuite) TestDeleteTargetPoolWithTargets() {
	pool, err := s.manager.CreateTargetPool(s.ctx, loadbalancer.CreateTargetPoolOptions{
		Name:     "busy-pool",
		Protocol: loadbalancer.ProtocolHTTP,
		Port:     8080,
	})
	s.Require().NoError(err)

	err = s.manager.AddTarget(s.ctx, pool.ID, loadbalancer.Target{Address: "10.0.0.1", Port: 8080})
	s.Require().NoError(err)

	// Should fail - pool has targets
	err = s.manager.DeleteTargetPool(s.ctx, pool.ID)
	s.True(errors.Is(err, loadbalancer.ErrTargetPoolInUse))
}

func (s *LoadBalancerManagerSuite) TestAddAndRemoveRule() {
	lb, err := s.manager.CreateLoadBalancer(s.ctx, loadbalancer.CreateLoadBalancerOptions{Name: "rule-test"})
	s.Require().NoError(err)

	pool, err := s.manager.CreateTargetPool(s.ctx, loadbalancer.CreateTargetPoolOptions{
		Name:     "api-pool",
		Protocol: loadbalancer.ProtocolHTTP,
		Port:     8080,
	})
	s.Require().NoError(err)

	listener, err := s.manager.CreateListener(s.ctx, loadbalancer.CreateListenerOptions{
		LoadBalancerID: lb.ID,
		Protocol:       loadbalancer.ProtocolHTTP,
		Port:           80,
		TargetPoolID:   pool.ID,
	})
	s.Require().NoError(err)

	rule, err := s.manager.AddRule(s.ctx, listener.ID, loadbalancer.Rule{
		Priority: 100,
		Conditions: []loadbalancer.RuleCondition{
			{Field: "path-pattern", Values: []string{"/api/*"}},
		},
		TargetPoolID: pool.ID,
	})
	s.Require().NoError(err)
	s.NotEmpty(rule.ID)
	s.Equal(100, rule.Priority)

	err = s.manager.RemoveRule(s.ctx, listener.ID, rule.ID)
	s.Require().NoError(err)
}

// TestLoadBalancerManagerSuite runs the test suite.
func TestLoadBalancerManagerSuite(t *testing.T) {
	suite.Run(t, new(LoadBalancerManagerSuite))
}

func TestSelectTargetRoundRobin(t *testing.T) {
	ctx := context.Background()
	mgr := memory.New()

	pool, err := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
		Name:      "rr-pool",
		Protocol:  loadbalancer.ProtocolHTTP,
		Port:      8080,
		Algorithm: loadbalancer.AlgorithmRoundRobin,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, addr := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		if err := mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: addr, Port: 8080}); err != nil {
			t.Fatal(err)
		}
	}

	seen := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		target, err := mgr.SelectTarget(ctx, pool.ID)
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, target.Address)
	}

	// Round-robin from pkg/algorithms/loadbalancing/roundrobin cycles in order.
	want := []string{
		"10.0.0.1", "10.0.0.2", "10.0.0.3",
		"10.0.0.1", "10.0.0.2", "10.0.0.3",
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("select %d: got %s, want %s (full=%v)", i, seen[i], want[i], seen)
		}
	}
}

func TestSelectTargetWeightedRoundRobin(t *testing.T) {
	ctx := context.Background()
	mgr := memory.New()

	pool, err := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
		Name:      "wrr-pool",
		Protocol:  loadbalancer.ProtocolHTTP,
		Port:      8080,
		Algorithm: loadbalancer.AlgorithmWeightedRoundRobin,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: "heavy", Port: 8080, Weight: 3}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: "light", Port: 8080, Weight: 1}); err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	for i := 0; i < 4; i++ {
		target, err := mgr.SelectTarget(ctx, pool.ID)
		if err != nil {
			t.Fatal(err)
		}
		counts[target.Address]++
	}

	if counts["heavy"] != 3 || counts["light"] != 1 {
		t.Fatalf("weighted selection counts = %v, want heavy=3 light=1", counts)
	}
}

func TestSelectTargetLeastConnections(t *testing.T) {
	ctx := context.Background()
	mgr := memory.New()

	pool, err := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
		Name:      "lc-pool",
		Protocol:  loadbalancer.ProtocolHTTP,
		Port:      8080,
		Algorithm: loadbalancer.AlgorithmLeastConnections,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: "a", Port: 8080}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: "b", Port: 8080}); err != nil {
		t.Fatal(err)
	}

	first, err := mgr.SelectTarget(ctx, pool.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Keep first connection open; next should prefer the other target.
	second, err := mgr.SelectTarget(ctx, pool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Address == second.Address {
		t.Fatalf("expected different targets under load, got %s twice", first.Address)
	}

	if err := mgr.ReleaseTarget(ctx, pool.ID, first.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSelectTargetEmptyPool(t *testing.T) {
	ctx := context.Background()
	mgr := memory.New()

	pool, err := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
		Name:     "empty",
		Protocol: loadbalancer.ProtocolHTTP,
		Port:     8080,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.SelectTarget(ctx, pool.ID)
	if !errors.Is(err, loadbalancer.ErrNoTargetsAvailable) {
		t.Fatalf("got %v, want ErrNoTargetsAvailable", err)
	}
}

func TestSelectTargetUnsupportedAlgorithm(t *testing.T) {
	ctx := context.Background()
	mgr := memory.New()

	_, err := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
		Name:      "iphash",
		Protocol:  loadbalancer.ProtocolHTTP,
		Port:      8080,
		Algorithm: loadbalancer.AlgorithmIPHash,
	})
	if !errors.Is(err, loadbalancer.ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", err)
	}
}

func TestSelectTargetRandom(t *testing.T) {
	ctx := context.Background()
	mgr := memory.New()

	pool, err := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
		Name:      "rand-pool",
		Protocol:  loadbalancer.ProtocolHTTP,
		Port:      8080,
		Algorithm: loadbalancer.AlgorithmRandom,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, addr := range []string{"10.0.0.1", "10.0.0.2"} {
		if err := mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: addr, Port: 8080}); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		target, err := mgr.SelectTarget(ctx, pool.ID)
		if err != nil {
			t.Fatal(err)
		}
		seen[target.Address] = true
	}
	if len(seen) < 2 {
		t.Fatalf("random selection only saw %v over 50 picks", seen)
	}
}

func TestTargetSelectorInterface(t *testing.T) {
	var _ loadbalancer.TargetSelector = memory.New()
	var _ loadbalancer.LoadBalancerManager = memory.New()
}
