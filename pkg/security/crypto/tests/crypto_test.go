package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/crypto"
	cryptomem "github.com/chris-alexander-pop/system-design-library/pkg/security/crypto/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type CryptoTestSuite struct {
	test.Suite
}

func (s *CryptoTestSuite) TestAESEncryptor() {
	key, err := crypto.GenerateAES256Key()
	s.NoError(err)

	enc, err := crypto.NewAESEncryptor(key)
	s.NoError(err)

	plaintext := "Hello, World!"
	ciphertext, err := enc.Encrypt([]byte(plaintext))
	s.NoError(err)
	s.NotEmpty(ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	s.NoError(err)
	s.Equal(plaintext, string(decrypted))
}

func (s *CryptoTestSuite) TestAESEncryptor_String() {
	key, err := crypto.GenerateAES256Key()
	s.NoError(err)

	enc, err := crypto.NewAESEncryptor(key)
	s.NoError(err)

	plaintext := "Sensitive Data"
	encoded, err := enc.EncryptString(plaintext)
	s.NoError(err)

	decrypted, err := enc.DecryptString(encoded)
	s.NoError(err)
	s.Equal(plaintext, decrypted)
}

func (s *CryptoTestSuite) TestAESEncryptor_InvalidKey() {
	_, err := crypto.NewAESEncryptor([]byte("short"))
	s.Error(err)
	s.True(errors.Is(err, crypto.ErrInvalidKey))
}

func (s *CryptoTestSuite) TestAESEncryptor_BadCiphertext() {
	key, err := crypto.GenerateAES256Key()
	s.NoError(err)
	enc, err := crypto.NewAESEncryptor(key)
	s.NoError(err)

	_, err = enc.Decrypt([]byte("tiny"))
	s.Error(err)
	s.True(errors.Is(err, crypto.ErrInvalidCiphertext))
}

func (s *CryptoTestSuite) TestInstrumentedEncryptor() {
	key, err := crypto.GenerateAES256Key()
	s.NoError(err)
	raw, err := crypto.NewAESEncryptor(key)
	s.NoError(err)

	enc := crypto.NewInstrumentedEncryptor(raw)
	ct, err := enc.Encrypt([]byte("instrumented"))
	s.NoError(err)
	pt, err := enc.Decrypt(ct)
	s.NoError(err)
	s.Equal("instrumented", string(pt))
}

func (s *CryptoTestSuite) TestEnvelope_MemoryKeyProvider() {
	master, err := crypto.GenerateAES256Key()
	s.NoError(err)
	kp, err := cryptomem.NewKeyProvider(master)
	s.NoError(err)

	env := crypto.NewEnvelopeEncryption(kp)
	payload, err := env.Encrypt(context.Background(), []byte("envelope-secret"))
	s.NoError(err)
	s.NotEmpty(payload.EncryptedData)
	s.Equal("AES-256-GCM", payload.Algorithm)

	out, err := env.Decrypt(context.Background(), payload)
	s.NoError(err)
	s.Equal("envelope-secret", string(out))
}

func TestCryptoSuite(t *testing.T) {
	test.Run(t, new(CryptoTestSuite))
}
