package azurekms

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	pkgkms "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/kms"
)

// EncryptDecryptAPI is the subset of the Azure Key Vault keys client used here.
// *azkeys.Client satisfies this interface.
type EncryptDecryptAPI interface {
	Encrypt(ctx context.Context, name string, version string, parameters azkeys.KeyOperationParameters, options *azkeys.EncryptOptions) (azkeys.EncryptResponse, error)
	Decrypt(ctx context.Context, name string, version string, parameters azkeys.KeyOperationParameters, options *azkeys.DecryptOptions) (azkeys.DecryptResponse, error)
}

// Config configures the Azure Key Vault KMS adapter.
type Config struct {
	// VaultURL is the Key Vault base URL (e.g. https://myvault.vault.azure.net/).
	VaultURL string `env:"AZURE_KEYVAULT_URL"`

	// Algorithm is the encryption algorithm (default RSA-OAEP-256).
	Algorithm string `env:"AZURE_KMS_ALGORITHM" env-default:"RSA-OAEP-256"`
}

// KeyManager implements pkgkms.KeyManager via Azure Key Vault.
type KeyManager struct {
	client    EncryptDecryptAPI
	algorithm azkeys.EncryptionAlgorithm
}

// Ensure KeyManager implements pkgkms.KeyManager.
var _ pkgkms.KeyManager = (*KeyManager)(nil)

// NewFromAPI wraps an existing EncryptDecryptAPI (SDK client or test double).
func NewFromAPI(api EncryptDecryptAPI, algorithm string) (*KeyManager, error) {
	if api == nil {
		return nil, errors.New(pkgkms.CodeInvalidArgument, "kms api client is required", nil)
	}
	var alg azkeys.EncryptionAlgorithm
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "", "RSA-OAEP-256":
		alg = azkeys.EncryptionAlgorithmRSAOAEP256
	case "RSA-OAEP":
		alg = azkeys.EncryptionAlgorithmRSAOAEP
	case "RSA1_5":
		alg = azkeys.EncryptionAlgorithmRSA15
	default:
		alg = azkeys.EncryptionAlgorithm(algorithm)
	}
	return &KeyManager{client: api, algorithm: alg}, nil
}

// New builds a KeyManager using DefaultAzureCredential against VaultURL.
func New(ctx context.Context, cfg Config) (*KeyManager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.VaultURL == "" {
		return nil, errors.New(pkgkms.CodeInvalidArgument, "azure key vault url is required", nil)
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.New(pkgkms.CodeUnavailable, "failed to create azure credential", err)
	}
	return NewFromCredential(cfg.VaultURL, cred, cfg.Algorithm)
}

// NewFromCredential builds a KeyManager with an explicit TokenCredential.
func NewFromCredential(vaultURL string, cred azcore.TokenCredential, algorithm string) (*KeyManager, error) {
	if vaultURL == "" {
		return nil, errors.New(pkgkms.CodeInvalidArgument, "azure key vault url is required", nil)
	}
	if cred == nil {
		return nil, errors.New(pkgkms.CodeInvalidArgument, "azure credential is required", nil)
	}
	client, err := azkeys.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, errors.New(pkgkms.CodeUnavailable, "failed to create azure keys client", err)
	}
	return NewFromAPI(client, algorithm)
}

// parseKeyID splits keyID into name and optional version.
// Accepts "name", "name/version", or a full key URL ending in /keys/name[/version].
func parseKeyID(keyID string) (name, version string, err error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return "", "", pkgkms.ErrInvalidArgument
	}
	if strings.Contains(keyID, "/keys/") {
		parts := strings.Split(keyID, "/keys/")
		rest := parts[len(parts)-1]
		segs := strings.Split(strings.Trim(rest, "/"), "/")
		if len(segs) == 0 || segs[0] == "" {
			return "", "", pkgkms.ErrInvalidArgument
		}
		name = segs[0]
		if len(segs) > 1 {
			version = segs[1]
		}
		return name, version, nil
	}
	segs := strings.Split(keyID, "/")
	name = segs[0]
	if len(segs) > 1 {
		version = segs[1]
	}
	return name, version, nil
}

// Encrypt encrypts plaintext under the given Key Vault key name (or name/version / URL).
func (m *KeyManager) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(plaintext) == 0 {
		return nil, pkgkms.ErrInvalidArgument
	}
	name, version, err := parseKeyID(keyID)
	if err != nil {
		return nil, err
	}

	out, err := m.client.Encrypt(ctx, name, version, azkeys.KeyOperationParameters{
		Algorithm: &m.algorithm,
		Value:     plaintext,
	}, nil)
	if err != nil {
		return nil, errors.New(pkgkms.CodeEncryptFailed, "azure kms encrypt failed", err)
	}
	if len(out.Result) == 0 {
		return nil, pkgkms.ErrEncryptFailed
	}
	return out.Result, nil
}

// Decrypt decrypts ciphertext previously produced by Encrypt.
func (m *KeyManager) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(ciphertext) == 0 {
		return nil, pkgkms.ErrInvalidArgument
	}
	name, version, err := parseKeyID(keyID)
	if err != nil {
		return nil, err
	}

	out, err := m.client.Decrypt(ctx, name, version, azkeys.KeyOperationParameters{
		Algorithm: &m.algorithm,
		Value:     ciphertext,
	}, nil)
	if err != nil {
		return nil, errors.New(pkgkms.CodeDecryptFailed, "azure kms decrypt failed", err)
	}
	if len(out.Result) == 0 {
		return nil, pkgkms.ErrDecryptFailed
	}
	return out.Result, nil
}
