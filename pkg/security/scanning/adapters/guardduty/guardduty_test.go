package guardduty_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning"
	gd "github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning/adapters/guardduty"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGD struct {
	ids      []string
	findings []types.Finding
	listErr  error
	getErr   error
}

func (f *fakeGD) ListFindings(ctx context.Context, params *guardduty.ListFindingsInput, _ ...func(*guardduty.Options)) (*guardduty.ListFindingsOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &guardduty.ListFindingsOutput{FindingIds: f.ids}, nil
}

func (f *fakeGD) GetFindings(ctx context.Context, params *guardduty.GetFindingsInput, _ ...func(*guardduty.Options)) (*guardduty.GetFindingsOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &guardduty.GetFindingsOutput{Findings: f.findings}, nil
}

type GuardDutySuite struct {
	test.Suite
}

func (s *GuardDutySuite) TestCleanWhenNoFindings() {
	api := &fakeGD{}
	sc, err := gd.NewFromAPI(api, "det-1")
	s.Require().NoError(err)

	rep, err := sc.Scan(s.Ctx, scanning.Resource{ID: "i-abc", Location: "i-abc"})
	s.Require().NoError(err)
	s.True(rep.Clean)
	s.Empty(rep.Threats)
}

func (s *GuardDutySuite) TestReportsMatchingThreat() {
	api := &fakeGD{
		ids: []string{"f1"},
		findings: []types.Finding{{
			Id:          aws.String("f1"),
			Type:        aws.String("UnauthorizedAccess:EC2/SSHBruteForce"),
			Title:       aws.String("SSH brute force on i-abc"),
			Description: aws.String("instance i-abc"),
			Resource: &types.Resource{
				InstanceDetails: &types.InstanceDetails{InstanceId: aws.String("i-abc")},
			},
		}},
	}
	sc, err := gd.NewFromAPI(api, "det-1")
	s.Require().NoError(err)

	rep, err := sc.Scan(s.Ctx, scanning.Resource{ID: "i-abc", Location: "i-abc", Type: "ec2"})
	s.Require().NoError(err)
	s.False(rep.Clean)
	s.Contains(rep.Threats, "UnauthorizedAccess:EC2/SSHBruteForce")
}

func (s *GuardDutySuite) TestInvalidResource() {
	sc, err := gd.NewFromAPI(&fakeGD{}, "det-1")
	s.Require().NoError(err)
	_, err = sc.Scan(s.Ctx, scanning.Resource{})
	s.Require().Error(err)
	s.True(errors.Is(err, scanning.ErrInvalidResource))
}

func (s *GuardDutySuite) TestNewFromAPIRequiresArgs() {
	_, err := gd.NewFromAPI(nil, "det")
	s.Require().Error(err)
	_, err = gd.NewFromAPI(&fakeGD{}, "")
	s.Require().Error(err)
}

func TestGuardDutySuite(t *testing.T) {
	test.Run(t, new(GuardDutySuite))
}

func TestImplementsScanner(t *testing.T) {
	sc, err := gd.NewFromAPI(&fakeGD{}, "d")
	require.NoError(t, err)
	var _ scanning.Scanner = sc
	assert.NotNil(t, sc)
}
