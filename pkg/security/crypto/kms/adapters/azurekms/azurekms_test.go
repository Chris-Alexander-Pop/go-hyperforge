package azurekms_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	pkgkms "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms/adapters/azurekms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type fakeAzureKMS struct {
	mu    sync.Mutex
	blobs map[string][]byte
	fail  error
}

func newFakeAzureKMS() *fakeAzureKMS {
	return &fakeAzureKMS{blobs: make(map[string][]byte)}
}

func (f *fakeAzureKMS) Encrypt(ctx context.Context, name string, version string, parameters azkeys.KeyOperationParameters, _ *azkeys.EncryptOptions) (azkeys.EncryptResponse, error) {
	if err := ctx.Err(); err != nil {
		return azkeys.EncryptResponse{}, err
	}
	if f.fail != nil {
		return azkeys.EncryptResponse{}, f.fail
	}
	if name == "" || len(parameters.Value) == 0 {
		return azkeys.EncryptResponse{}, errors.New("invalid encrypt")
	}
	ct := append([]byte("az:"+name+":"), parameters.Value...)
	f.mu.Lock()
	f.blobs[string(ct)] = append([]byte(nil), parameters.Value...)
	f.mu.Unlock()
	return azkeys.EncryptResponse{
		KeyOperationResult: azkeys.KeyOperationResult{Result: ct},
	}, nil
}

func (f *fakeAzureKMS) Decrypt(ctx context.Context, name string, version string, parameters azkeys.KeyOperationParameters, _ *azkeys.DecryptOptions) (azkeys.DecryptResponse, error) {
	if err := ctx.Err(); err != nil {
		return azkeys.DecryptResponse{}, err
	}
	if f.fail != nil {
		return azkeys.DecryptResponse{}, f.fail
	}
	if len(parameters.Value) == 0 {
		return azkeys.DecryptResponse{}, errors.New("invalid decrypt")
	}
	f.mu.Lock()
	pt, ok := f.blobs[string(parameters.Value)]
	f.mu.Unlock()
	if !ok {
		return azkeys.DecryptResponse{}, errors.New("not found")
	}
	return azkeys.DecryptResponse{
		KeyOperationResult: azkeys.KeyOperationResult{Result: append([]byte(nil), pt...)},
	}, nil
}

type AzureKMSSuite struct {
	test.Suite
	fake *fakeAzureKMS
	mgr  *azurekms.KeyManager
}

func (s *AzureKMSSuite) SetupTest() {
	s.Suite.SetupTest()
	s.fake = newFakeAzureKMS()
	var err error
	s.mgr, err = azurekms.NewFromAPI(s.fake, "RSA-OAEP-256")
	s.Require().NoError(err)
}

func (s *AzureKMSSuite) TestEncryptDecrypt() {
	ct, err := s.mgr.Encrypt(s.Ctx, "my-key", []byte("hello"))
	s.Require().NoError(err)
	s.NotEqual([]byte("hello"), ct)

	pt, err := s.mgr.Decrypt(s.Ctx, "my-key", ct)
	s.Require().NoError(err)
	s.Equal([]byte("hello"), pt)
}

func (s *AzureKMSSuite) TestEncryptKeyURL() {
	ct, err := s.mgr.Encrypt(s.Ctx, "https://vault.vault.azure.net/keys/k1/abc", []byte("x"))
	s.Require().NoError(err)
	pt, err := s.mgr.Decrypt(s.Ctx, "k1/abc", ct)
	s.Require().NoError(err)
	s.Equal([]byte("x"), pt)
}

func (s *AzureKMSSuite) TestEncryptEmptyKey() {
	_, err := s.mgr.Encrypt(s.Ctx, "", []byte("x"))
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *AzureKMSSuite) TestEncryptEmptyPlaintext() {
	_, err := s.mgr.Encrypt(s.Ctx, "key", nil)
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *AzureKMSSuite) TestDecryptEmpty() {
	_, err := s.mgr.Decrypt(s.Ctx, "key", nil)
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *AzureKMSSuite) TestEncryptAPIFailure() {
	s.fake.fail = errors.New("kv down")
	_, err := s.mgr.Encrypt(s.Ctx, "key", []byte("x"))
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, pkgkms.CodeEncryptFailed))
}

func (s *AzureKMSSuite) TestNewFromAPINil() {
	_, err := azurekms.NewFromAPI(nil, "")
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, pkgkms.CodeInvalidArgument))
}

func (s *AzureKMSSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.mgr.Encrypt(ctx, "key", []byte("x"))
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestAzureKMSSuite(t *testing.T) {
	test.Run(t, new(AzureKMSSuite))
}
