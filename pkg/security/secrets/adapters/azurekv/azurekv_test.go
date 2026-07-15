package azurekv_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets/adapters/azurekv"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type fakeAzureSecrets struct {
	mu   sync.Mutex
	data map[string]string
	fail error
}

func newFakeAzureSecrets() *fakeAzureSecrets {
	return &fakeAzureSecrets{data: make(map[string]string)}
}

func (f *fakeAzureSecrets) GetSecret(ctx context.Context, name string, version string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	if err := ctx.Err(); err != nil {
		return azsecrets.GetSecretResponse{}, err
	}
	if f.fail != nil {
		return azsecrets.GetSecretResponse{}, f.fail
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.data[name]
	if !ok {
		return azsecrets.GetSecretResponse{}, &azcore.ResponseError{StatusCode: http.StatusNotFound, ErrorCode: "SecretNotFound"}
	}
	return azsecrets.GetSecretResponse{Secret: azsecrets.Secret{Value: &v}}, nil
}

func (f *fakeAzureSecrets) SetSecret(ctx context.Context, name string, parameters azsecrets.SetSecretParameters, _ *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
	if err := ctx.Err(); err != nil {
		return azsecrets.SetSecretResponse{}, err
	}
	if f.fail != nil {
		return azsecrets.SetSecretResponse{}, f.fail
	}
	if parameters.Value == nil {
		return azsecrets.SetSecretResponse{}, errors.New("missing value")
	}
	f.mu.Lock()
	f.data[name] = *parameters.Value
	f.mu.Unlock()
	return azsecrets.SetSecretResponse{Secret: azsecrets.Secret{Value: parameters.Value}}, nil
}

func (f *fakeAzureSecrets) DeleteSecret(ctx context.Context, name string, _ *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error) {
	if err := ctx.Err(); err != nil {
		return azsecrets.DeleteSecretResponse{}, err
	}
	if f.fail != nil {
		return azsecrets.DeleteSecretResponse{}, f.fail
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[name]; !ok {
		return azsecrets.DeleteSecretResponse{}, &azcore.ResponseError{StatusCode: http.StatusNotFound, ErrorCode: "SecretNotFound"}
	}
	delete(f.data, name)
	return azsecrets.DeleteSecretResponse{}, nil
}

type AzureKVSuite struct {
	test.Suite
	fake *fakeAzureSecrets
	mgr  *azurekv.Manager
}

func (s *AzureKVSuite) SetupTest() {
	s.Suite.SetupTest()
	s.fake = newFakeAzureSecrets()
	var err error
	s.mgr, err = azurekv.NewFromAPI(s.fake)
	s.Require().NoError(err)
}

func (s *AzureKVSuite) TestSetGet() {
	err := s.mgr.Set(s.Ctx, "db-pass", "s3cret")
	s.Require().NoError(err)
	v, err := s.mgr.Get(s.Ctx, "db-pass")
	s.Require().NoError(err)
	s.Equal("s3cret", v)
}

func (s *AzureKVSuite) TestGetURL() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "tok", "v1"))
	v, err := s.mgr.Get(s.Ctx, "https://vault.vault.azure.net/secrets/tok/abc")
	s.Require().NoError(err)
	s.Equal("v1", v)
}

func (s *AzureKVSuite) TestGetNotFound() {
	_, err := s.mgr.Get(s.Ctx, "missing")
	s.Require().Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *AzureKVSuite) TestDelete() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "tmp", "x"))
	s.Require().NoError(s.mgr.Delete(s.Ctx, "tmp"))
	_, err := s.mgr.Get(s.Ctx, "tmp")
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *AzureKVSuite) TestDeleteNotFound() {
	err := s.mgr.Delete(s.Ctx, "missing")
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *AzureKVSuite) TestRotate() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "tok", "old"))
	nv, err := s.mgr.Rotate(s.Ctx, "tok", "new")
	s.Require().NoError(err)
	s.Equal("new", nv)
	v, err := s.mgr.Get(s.Ctx, "tok")
	s.Require().NoError(err)
	s.Equal("new", v)
}

func (s *AzureKVSuite) TestEmptyName() {
	_, err := s.mgr.Get(s.Ctx, "")
	s.True(errors.Is(err, secrets.ErrInvalidArgument))
}

func (s *AzureKVSuite) TestNewFromAPINil() {
	_, err := azurekv.NewFromAPI(nil)
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, secrets.CodeInvalidArgument))
}

func (s *AzureKVSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.mgr.Get(ctx, "x")
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func (s *AzureKVSuite) TestImplementsSecretManager() {
	var _ secrets.SecretManager = s.mgr
}

func TestAzureKVSuite(t *testing.T) {
	test.Run(t, new(AzureKVSuite))
}
