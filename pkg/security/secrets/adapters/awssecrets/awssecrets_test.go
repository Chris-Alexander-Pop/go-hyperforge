package awssecrets_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets/adapters/awssecrets"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type fakeSM struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeSM() *fakeSM {
	return &fakeSM{data: make(map[string]string)}
}

func (f *fakeSM) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.data[aws.ToString(params.SecretId)]
	if !ok {
		return nil, &smtypes.ResourceNotFoundException{Message: aws.String("not found")}
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(v)}, nil
}

func (f *fakeSM) PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	id := aws.ToString(params.SecretId)
	if _, ok := f.data[id]; !ok {
		return nil, &smtypes.ResourceNotFoundException{Message: aws.String("not found")}
	}
	f.data[id] = aws.ToString(params.SecretString)
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (f *fakeSM) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	name := aws.ToString(params.Name)
	f.data[name] = aws.ToString(params.SecretString)
	return &secretsmanager.CreateSecretOutput{Name: params.Name}, nil
}

type AWSSecretsSuite struct {
	test.Suite
	mgr *awssecrets.Manager
}

func (s *AWSSecretsSuite) SetupTest() {
	s.Suite.SetupTest()
	var err error
	s.mgr, err = awssecrets.NewFromAPI(newFakeSM())
	s.Require().NoError(err)
}

func (s *AWSSecretsSuite) TestSetGet() {
	err := s.mgr.Set(s.Ctx, "db/pass", "s3cret")
	s.Require().NoError(err)
	v, err := s.mgr.Get(s.Ctx, "db/pass")
	s.Require().NoError(err)
	s.Equal("s3cret", v)
}

func (s *AWSSecretsSuite) TestGetNotFound() {
	_, err := s.mgr.Get(s.Ctx, "missing")
	s.Require().Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *AWSSecretsSuite) TestRotate() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "tok", "old"))
	nv, err := s.mgr.Rotate(s.Ctx, "tok", "new")
	s.Require().NoError(err)
	s.Equal("new", nv)
	v, err := s.mgr.Get(s.Ctx, "tok")
	s.Require().NoError(err)
	s.Equal("new", v)
}

func (s *AWSSecretsSuite) TestEmptyName() {
	_, err := s.mgr.Get(s.Ctx, "")
	s.True(errors.Is(err, secrets.ErrInvalidArgument))
}

func TestAWSSecretsSuite(t *testing.T) {
	test.Run(t, new(AWSSecretsSuite))
}
