package aws_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf"
	wafaws "github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf/adapters/aws"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type fakeIPSet struct {
	mu        sync.Mutex
	addresses []string
	lock      string
	fail      error
}

func newFakeIPSet() *fakeIPSet {
	return &fakeIPSet{lock: "tok-1", addresses: nil}
}

func (f *fakeIPSet) GetIPSet(ctx context.Context, params *wafv2.GetIPSetInput, _ ...func(*wafv2.Options)) (*wafv2.GetIPSetOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.fail != nil {
		return nil, f.fail
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	addrs := append([]string{}, f.addresses...)
	return &wafv2.GetIPSetOutput{
		IPSet: &types.IPSet{
			Id:        params.Id,
			Name:      params.Name,
			Addresses: addrs,
		},
		LockToken: aws.String(f.lock),
	}, nil
}

func (f *fakeIPSet) UpdateIPSet(ctx context.Context, params *wafv2.UpdateIPSetInput, _ ...func(*wafv2.Options)) (*wafv2.UpdateIPSetOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.fail != nil {
		return nil, f.fail
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.addresses = append([]string{}, params.Addresses...)
	f.lock = "tok-2"
	return &wafv2.UpdateIPSetOutput{NextLockToken: aws.String(f.lock)}, nil
}

type AWSWAFSuite struct {
	test.Suite
	fake *fakeIPSet
	mgr  *wafaws.Manager
}

func (s *AWSWAFSuite) SetupTest() {
	s.Suite.SetupTest()
	s.fake = newFakeIPSet()
	var err error
	s.mgr, err = wafaws.NewFromAPI(s.fake, wafaws.Config{
		IPSetID:   "id-1",
		IPSetName: "blocked",
		Scope:     "REGIONAL",
	})
	s.Require().NoError(err)
}

func (s *AWSWAFSuite) TestBlockAllowList() {
	s.Require().NoError(s.mgr.BlockIP(s.Ctx, "203.0.113.10", "abuse"))
	rules, err := s.mgr.GetRules(s.Ctx)
	s.Require().NoError(err)
	s.Require().Len(rules, 1)
	s.Equal("203.0.113.10", rules[0].IP)
	s.Equal("block", rules[0].Action)

	s.Require().NoError(s.mgr.AllowIP(s.Ctx, "203.0.113.10"))
	rules, err = s.mgr.GetRules(s.Ctx)
	s.Require().NoError(err)
	s.Empty(rules)
}

func (s *AWSWAFSuite) TestAllowNotFound() {
	err := s.mgr.AllowIP(s.Ctx, "203.0.113.99")
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, waf.ErrNotFound))
}

func (s *AWSWAFSuite) TestEmptyIP() {
	err := s.mgr.BlockIP(s.Ctx, "", "x")
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, waf.ErrInvalidRule))
}

func (s *AWSWAFSuite) TestNewFromAPINil() {
	_, err := wafaws.NewFromAPI(nil, wafaws.Config{IPSetID: "a", IPSetName: "b"})
	s.Require().Error(err)
}

func (s *AWSWAFSuite) TestAPIFailure() {
	s.fake.fail = errors.New("down")
	err := s.mgr.BlockIP(s.Ctx, "1.1.1.1", "x")
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, waf.CodeUnavailable))
}

func (s *AWSWAFSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	err := s.mgr.BlockIP(ctx, "1.1.1.1", "x")
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestAWSWAFSuite(t *testing.T) {
	test.Run(t, new(AWSWAFSuite))
}
