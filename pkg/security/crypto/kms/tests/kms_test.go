package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type KMSTestSuite struct {
	test.Suite
	manager kms.KeyManager
}

func (s *KMSTestSuite) SetupTest() {
	s.Suite.SetupTest()
	var err error
	s.manager, err = memory.New("")
	if err != nil {
		panic(err)
	}
}

func (s *KMSTestSuite) TestEncryptDecrypt() {
	keyID := "key-1"
	plaintext := []byte("secret data")

	ciphertext, err := s.manager.Encrypt(s.Ctx, keyID, plaintext)
	s.NoError(err)
	s.NotNil(ciphertext)
	s.NotEqual(plaintext, ciphertext)

	decrypted, err := s.manager.Decrypt(s.Ctx, keyID, ciphertext)
	s.NoError(err)
	s.Equal(plaintext, decrypted)
}

func (s *KMSTestSuite) TestResilientEncryptDecrypt() {
	inner, err := memory.New("")
	s.Require().NoError(err)
	mgr := kms.NewResilientKeyManager(inner, kms.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	ct, err := mgr.Encrypt(s.Ctx, "k", []byte("hello"))
	s.NoError(err)
	pt, err := mgr.Decrypt(s.Ctx, "k", ct)
	s.NoError(err)
	s.Equal([]byte("hello"), pt)
}

func TestKMSSuite(t *testing.T) {
	test.Run(t, new(KMSTestSuite))
}
