package test_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type suiteSelfTest struct {
	test.Suite
	setupCalls int
}

func (s *suiteSelfTest) SetupTest() {
	s.Suite.SetupTest()
	s.setupCalls++
}

func (s *suiteSelfTest) TestCtxIsBackground() {
	s.NotNil(s.Ctx)
	s.NoError(s.Ctx.Err())
	s.Equal(context.Background(), s.Ctx)
}

func (s *suiteSelfTest) TestAssertHelper() {
	a := s.Assert()
	s.NotNil(a)
	a.True(true)
}

func (s *suiteSelfTest) TestNewSuiteFactory() {
	ns := test.NewSuite()
	s.NotNil(ns)
	ns.SetupTest()
	s.NotNil(ns.Ctx)
}

func (s *suiteSelfTest) TestSetupRunsPerMethod() {
	s.GreaterOrEqual(s.setupCalls, 1)
}

func TestSuiteSelf(t *testing.T) {
	test.Run(t, new(suiteSelfTest))
}

func TestRunInvokesTestifySuite(t *testing.T) {
	type tiny struct {
		suite.Suite
	}
	ts := &tiny{}
	test.Run(t, ts)
	assert.NotNil(t, ts)
}

func TestStartPostgresShortSkip(t *testing.T) {
	if !testing.Short() {
		t.Skip("only asserts Short skip behavior")
	}
	test.StartPostgres(t)
	t.Fatal("expected StartPostgres to skip in short mode")
}

func TestStartRedisShortSkip(t *testing.T) {
	if !testing.Short() {
		t.Skip("only asserts Short skip behavior")
	}
	test.StartRedis(t)
	t.Fatal("expected StartRedis to skip in short mode")
}
