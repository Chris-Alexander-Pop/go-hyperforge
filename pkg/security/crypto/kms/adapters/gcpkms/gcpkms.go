package gcpkms

import (
	"context"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	pkgkms "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
)

// EncryptDecryptAPI is the subset of the GCP KMS client used by this adapter.
// *kms.KeyManagementClient satisfies this interface.
type EncryptDecryptAPI interface {
	Encrypt(ctx context.Context, req *kmspb.EncryptRequest, opts ...gax.CallOption) (*kmspb.EncryptResponse, error)
	Decrypt(ctx context.Context, req *kmspb.DecryptRequest, opts ...gax.CallOption) (*kmspb.DecryptResponse, error)
	Close() error
}

// Config configures the GCP KMS adapter.
type Config struct {
	// CredentialsFile is an optional path to a service account JSON key.
	CredentialsFile string `env:"GOOGLE_APPLICATION_CREDENTIALS"`

	// Endpoint overrides the KMS API endpoint (tests / regional).
	Endpoint string `env:"GCP_KMS_ENDPOINT"`
}

// KeyManager implements pkgkms.KeyManager via GCP Cloud KMS.
type KeyManager struct {
	client EncryptDecryptAPI
}

// Ensure KeyManager implements pkgkms.KeyManager.
var _ pkgkms.KeyManager = (*KeyManager)(nil)

// NewFromAPI wraps an existing EncryptDecryptAPI (SDK client or test double).
func NewFromAPI(api EncryptDecryptAPI) (*KeyManager, error) {
	if api == nil {
		return nil, errors.New(pkgkms.CodeInvalidArgument, "kms api client is required", nil)
	}
	return &KeyManager{client: api}, nil
}

// New builds a KeyManager from GCP client options.
func New(ctx context.Context, cfg Config) (*KeyManager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}
	if cfg.Endpoint != "" {
		opts = append(opts, option.WithEndpoint(cfg.Endpoint))
	}
	client, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, errors.New(pkgkms.CodeUnavailable, "failed to create gcp kms client", err)
	}
	return NewFromAPI(client)
}

// Encrypt encrypts plaintext under the given CryptoKey resource name.
// keyID should be projects/*/locations/*/keyRings/*/cryptoKeys/*.
func (m *KeyManager) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if keyID == "" {
		return nil, pkgkms.ErrInvalidArgument
	}
	if len(plaintext) == 0 {
		return nil, pkgkms.ErrInvalidArgument
	}

	out, err := m.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      keyID,
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, errors.New(pkgkms.CodeEncryptFailed, "gcp kms encrypt failed", err)
	}
	if out == nil || len(out.Ciphertext) == 0 {
		return nil, pkgkms.ErrEncryptFailed
	}
	return out.Ciphertext, nil
}

// Decrypt decrypts ciphertext previously produced by Encrypt.
// keyID should be the same CryptoKey resource name used for Encrypt.
func (m *KeyManager) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if keyID == "" {
		return nil, pkgkms.ErrInvalidArgument
	}
	if len(ciphertext) == 0 {
		return nil, pkgkms.ErrInvalidArgument
	}

	out, err := m.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       keyID,
		Ciphertext: ciphertext,
	})
	if err != nil {
		return nil, errors.New(pkgkms.CodeDecryptFailed, "gcp kms decrypt failed", err)
	}
	if out == nil || len(out.Plaintext) == 0 {
		return nil, pkgkms.ErrDecryptFailed
	}
	return out.Plaintext, nil
}

// Close closes the underlying client when it supports Close.
func (m *KeyManager) Close() error {
	return m.client.Close()
}
