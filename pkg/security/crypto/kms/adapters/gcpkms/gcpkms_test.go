package gcpkms_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	pkgkms "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms/adapters/gcpkms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"github.com/googleapis/gax-go/v2"
)

type fakeGCPKMS struct {
	mu    sync.Mutex
	blobs map[string][]byte
	fail  error
}

func newFakeGCPKMS() *fakeGCPKMS {
	return &fakeGCPKMS{blobs: make(map[string][]byte)}
}

func (f *fakeGCPKMS) Encrypt(ctx context.Context, req *kmspb.EncryptRequest, _ ...gax.CallOption) (*kmspb.EncryptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.fail != nil {
		return nil, f.fail
	}
	if req == nil || req.Name == "" || len(req.Plaintext) == 0 {
		return nil, errors.New("invalid encrypt")
	}
	ct := append([]byte("gcp:"), req.Plaintext...)
	f.mu.Lock()
	f.blobs[string(ct)] = append([]byte(nil), req.Plaintext...)
	f.mu.Unlock()
	return &kmspb.EncryptResponse{Ciphertext: ct, Name: req.Name}, nil
}

func (f *fakeGCPKMS) Decrypt(ctx context.Context, req *kmspb.DecryptRequest, _ ...gax.CallOption) (*kmspb.DecryptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.fail != nil {
		return nil, f.fail
	}
	if req == nil || len(req.Ciphertext) == 0 {
		return nil, errors.New("invalid decrypt")
	}
	f.mu.Lock()
	pt, ok := f.blobs[string(req.Ciphertext)]
	f.mu.Unlock()
	if !ok {
		return nil, errors.New("not found")
	}
	return &kmspb.DecryptResponse{Plaintext: append([]byte(nil), pt...)}, nil
}

func (f *fakeGCPKMS) Close() error { return nil }

type GCPKMSSuite struct {
	test.Suite
	fake *fakeGCPKMS
	mgr  *gcpkms.KeyManager
}

func (s *GCPKMSSuite) SetupTest() {
	s.Suite.SetupTest()
	s.fake = newFakeGCPKMS()
	var err error
	s.mgr, err = gcpkms.NewFromAPI(s.fake)
	s.Require().NoError(err)
}

func (s *GCPKMSSuite) TestEncryptDecrypt() {
	key := "projects/p/locations/global/keyRings/r/cryptoKeys/k"
	ct, err := s.mgr.Encrypt(s.Ctx, key, []byte("hello"))
	s.Require().NoError(err)
	s.NotEqual([]byte("hello"), ct)

	pt, err := s.mgr.Decrypt(s.Ctx, key, ct)
	s.Require().NoError(err)
	s.Equal([]byte("hello"), pt)
}

func (s *GCPKMSSuite) TestEncryptEmptyKey() {
	_, err := s.mgr.Encrypt(s.Ctx, "", []byte("x"))
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *GCPKMSSuite) TestEncryptEmptyPlaintext() {
	_, err := s.mgr.Encrypt(s.Ctx, "key", nil)
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *GCPKMSSuite) TestDecryptEmpty() {
	_, err := s.mgr.Decrypt(s.Ctx, "key", nil)
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *GCPKMSSuite) TestEncryptAPIFailure() {
	s.fake.fail = errors.New("kms down")
	_, err := s.mgr.Encrypt(s.Ctx, "key", []byte("x"))
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, pkgkms.CodeEncryptFailed))
}

func (s *GCPKMSSuite) TestNewFromAPINil() {
	_, err := gcpkms.NewFromAPI(nil)
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, pkgkms.CodeInvalidArgument))
}

func (s *GCPKMSSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.mgr.Encrypt(ctx, "key", []byte("x"))
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestGCPKMSSuite(t *testing.T) {
	test.Run(t, new(GCPKMSSuite))
}
