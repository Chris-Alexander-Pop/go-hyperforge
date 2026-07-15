package tests

import (
	"testing"

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

func TestWAFSuite(t *testing.T) {
	test.Run(t, new(WAFTestSuite))
}
