package awskms_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	pkgkms "github.com/chris-alexander-pop/system-design-library/pkg/security/crypto/kms"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/crypto/kms/adapters/awskms"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

// fakeKMS is an in-process EncryptDecryptAPI for unit tests.
type fakeKMS struct {
	mu    sync.Mutex
	blobs map[string][]byte // ciphertext -> plaintext
	fail  error
}

func newFakeKMS() *fakeKMS {
	return &fakeKMS{blobs: make(map[string][]byte)}
}

func (f *fakeKMS) Encrypt(ctx context.Context, params *kms.EncryptInput, _ ...func(*kms.Options)) (*kms.EncryptOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.fail != nil {
		return nil, f.fail
	}
	if params == nil || params.KeyId == nil || *params.KeyId == "" || len(params.Plaintext) == 0 {
		return nil, errors.New("invalid encrypt input")
	}
	ct := append([]byte("enc:"), params.Plaintext...)
	f.mu.Lock()
	f.blobs[string(ct)] = append([]byte(nil), params.Plaintext...)
	f.mu.Unlock()
	return &kms.EncryptOutput{CiphertextBlob: ct, KeyId: params.KeyId}, nil
}

func (f *fakeKMS) Decrypt(ctx context.Context, params *kms.DecryptInput, _ ...func(*kms.Options)) (*kms.DecryptOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.fail != nil {
		return nil, f.fail
	}
	if params == nil || len(params.CiphertextBlob) == 0 {
		return nil, errors.New("invalid decrypt input")
	}
	f.mu.Lock()
	pt, ok := f.blobs[string(params.CiphertextBlob)]
	f.mu.Unlock()
	if !ok {
		return nil, errors.New("ciphertext not found")
	}
	return &kms.DecryptOutput{Plaintext: append([]byte(nil), pt...)}, nil
}

type AWSKMSSuite struct {
	test.Suite
	fake *fakeKMS
	mgr  *awskms.KeyManager
}

func (s *AWSKMSSuite) SetupTest() {
	s.Suite.SetupTest()
	s.fake = newFakeKMS()
	var err error
	s.mgr, err = awskms.NewFromAPI(s.fake)
	s.Require().NoError(err)
}

func (s *AWSKMSSuite) TestEncryptDecrypt() {
	ct, err := s.mgr.Encrypt(s.Ctx, "alias/test", []byte("hello"))
	s.Require().NoError(err)
	s.NotEqual([]byte("hello"), ct)

	pt, err := s.mgr.Decrypt(s.Ctx, "alias/test", ct)
	s.Require().NoError(err)
	s.Equal([]byte("hello"), pt)
}

func (s *AWSKMSSuite) TestEncryptEmptyKey() {
	_, err := s.mgr.Encrypt(s.Ctx, "", []byte("x"))
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *AWSKMSSuite) TestEncryptEmptyPlaintext() {
	_, err := s.mgr.Encrypt(s.Ctx, "key", nil)
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *AWSKMSSuite) TestDecryptEmpty() {
	_, err := s.mgr.Decrypt(s.Ctx, "key", nil)
	s.Require().Error(err)
	s.True(pkgerrors.Is(err, pkgkms.ErrInvalidArgument))
}

func (s *AWSKMSSuite) TestEncryptAPIFailure() {
	s.fake.fail = errors.New("kms down")
	_, err := s.mgr.Encrypt(s.Ctx, "key", []byte("x"))
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, pkgkms.CodeEncryptFailed))
}

func (s *AWSKMSSuite) TestNewFromAPINil() {
	_, err := awskms.NewFromAPI(nil)
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, pkgkms.CodeInvalidArgument))
}

func (s *AWSKMSSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.mgr.Encrypt(ctx, "key", []byte("x"))
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestAWSKMSSuite(t *testing.T) {
	test.Run(t, new(AWSKMSSuite))
}
