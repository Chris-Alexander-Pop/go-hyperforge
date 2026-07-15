package awskms

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	pkgkms "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
)

// EncryptDecryptAPI is the subset of the AWS KMS client used by this adapter.
// *kms.Client from aws-sdk-go-v2/service/kms satisfies this interface.
type EncryptDecryptAPI interface {
	Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

// Config configures the AWS KMS adapter.
type Config struct {
	// Region is the AWS region (required for New).
	Region string `env:"AWS_REGION" env-default:"us-east-1"`

	// AccessKeyID / SecretAccessKey are optional static credentials.
	// When empty, the default AWS credential chain is used.
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`

	// Endpoint overrides the KMS endpoint (LocalStack / tests).
	Endpoint string `env:"AWS_KMS_ENDPOINT"`
}

// KeyManager implements pkgkms.KeyManager via AWS KMS.
type KeyManager struct {
	client EncryptDecryptAPI
}

// Ensure KeyManager implements pkgkms.KeyManager.
var _ pkgkms.KeyManager = (*KeyManager)(nil)

// NewFromAPI wraps an existing EncryptDecryptAPI (SDK client or test double).
func NewFromAPI(api EncryptDecryptAPI) (*KeyManager, error) {
	if api == nil {
		return nil, pkgerrors.New(pkgkms.CodeInvalidArgument, "kms api client is required", nil)
	}
	return &KeyManager{client: api}, nil
}

// New builds a KeyManager from AWS SDK config.
func New(ctx context.Context, cfg Config) (*KeyManager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, pkgerrors.New(pkgkms.CodeInvalidArgument, "aws region is required", nil)
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.New(pkgkms.CodeUnavailable, "failed to load aws config", err)
	}

	client := kms.NewFromConfig(awsCfg, func(o *kms.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return NewFromAPI(client)
}

// Encrypt encrypts plaintext under the given KMS key ID / ARN / alias.
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

	out, err := m.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:     aws.String(keyID),
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, pkgerrors.New(pkgkms.CodeEncryptFailed, "aws kms encrypt failed", err)
	}
	if out == nil || len(out.CiphertextBlob) == 0 {
		return nil, pkgkms.ErrEncryptFailed
	}
	return out.CiphertextBlob, nil
}

// Decrypt decrypts ciphertext previously produced by Encrypt.
func (m *KeyManager) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(ciphertext) == 0 {
		return nil, pkgkms.ErrInvalidArgument
	}

	in := &kms.DecryptInput{CiphertextBlob: ciphertext}
	// KeyId is optional for Decrypt when the blob embeds key metadata,
	// but pass it when provided for explicit key pinning.
	if keyID != "" {
		in.KeyId = aws.String(keyID)
	}

	out, err := m.client.Decrypt(ctx, in)
	if err != nil {
		return nil, pkgerrors.New(pkgkms.CodeDecryptFailed, "aws kms decrypt failed", err)
	}
	if out == nil || len(out.Plaintext) == 0 {
		return nil, pkgkms.ErrDecryptFailed
	}
	return out.Plaintext, nil
}
