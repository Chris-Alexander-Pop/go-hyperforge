package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type WAFTestSuite struct {
	test.Suite
	manager waf.Manager
}

func (s *WAFTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.manager = memory.New()
}

func (s *WAFTestSuite) TestBlockAndAllow() {
	ip := "192.168.1.1"

	err := s.manager.BlockIP(s.Ctx, ip, "bad behavior")
	s.NoError(err)

	rules, err := s.manager.GetRules(s.Ctx)
	s.NoError(err)
	s.NotEmpty(rules)
	s.Equal(ip, rules[0].IP)

	err = s.manager.AllowIP(s.Ctx, ip)
	s.NoError(err)

	rules, err = s.manager.GetRules(s.Ctx)
	s.NoError(err)
	s.Empty(rules)
}

func (s *WAFTestSuite) TestResilientBlockAndAllow() {
	mgr := waf.NewResilientManager(memory.New(), waf.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	s.NoError(mgr.BlockIP(s.Ctx, "10.0.0.1", "test"))
	rules, err := mgr.GetRules(s.Ctx)
	s.NoError(err)
	s.Len(rules, 1)
	s.NoError(mgr.AllowIP(s.Ctx, "10.0.0.1"))
}

func TestWAFSuite(t *testing.T) {
	test.Run(t, new(WAFTestSuite))
}
