// Package azurekv implements secrets.SecretManager via Azure Key Vault secrets.
//
// Get/Set/Delete wrap GetSecret, SetSecret, and DeleteSecret.
// Inject SecretsAPI via NewFromAPI for tests; New builds the SDK client.
package azurekv

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
)

// SecretsAPI is the Azure Key Vault secrets surface used by this adapter.
// *azsecrets.Client satisfies this interface.
type SecretsAPI interface {
	GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
	SetSecret(ctx context.Context, name string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
	DeleteSecret(ctx context.Context, name string, options *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error)
}

// Config configures the Azure Key Vault secrets adapter.
type Config struct {
	// VaultURL is the Key Vault base URL (e.g. https://myvault.vault.azure.net/).
	VaultURL string `env:"AZURE_KEYVAULT_URL"`
}

// Manager implements secrets.SecretManager via Azure Key Vault.
// Delete is an additional Azure-specific method beyond the SecretManager interface.
type Manager struct {
	client SecretsAPI
}

// Ensure Manager implements secrets.SecretManager.
var _ secrets.SecretManager = (*Manager)(nil)

// NewFromAPI wraps an existing SecretsAPI (SDK client or test double).
func NewFromAPI(api SecretsAPI) (*Manager, error) {
	if api == nil {
		return nil, pkgerrors.New(secrets.CodeInvalidArgument, "azure key vault secrets api is required", nil)
	}
	return &Manager{client: api}, nil
}

// New builds a Manager using DefaultAzureCredential against VaultURL.
func New(ctx context.Context, cfg Config) (*Manager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.VaultURL == "" {
		return nil, pkgerrors.New(secrets.CodeInvalidArgument, "azure key vault url is required", nil)
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, pkgerrors.New(secrets.CodeUnavailable, "failed to create azure credential", err)
	}
	return NewFromCredential(cfg.VaultURL, cred)
}

// NewFromCredential builds a Manager with an explicit TokenCredential.
func NewFromCredential(vaultURL string, cred azcore.TokenCredential) (*Manager, error) {
	if vaultURL == "" {
		return nil, pkgerrors.New(secrets.CodeInvalidArgument, "azure key vault url is required", nil)
	}
	if cred == nil {
		return nil, pkgerrors.New(secrets.CodeInvalidArgument, "azure credential is required", nil)
	}
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, pkgerrors.New(secrets.CodeUnavailable, "failed to create azure secrets client", err)
	}
	return NewFromAPI(client)
}

// parseSecretName splits name into secret name and optional version.
// Accepts "name", "name/version", or a full secret URL ending in /secrets/name[/version].
func parseSecretName(name string) (secretName, version string, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", secrets.ErrInvalidArgument
	}
	if strings.Contains(name, "/secrets/") {
		parts := strings.Split(name, "/secrets/")
		rest := parts[len(parts)-1]
		segs := strings.Split(strings.Trim(rest, "/"), "/")
		if len(segs) == 0 || segs[0] == "" {
			return "", "", secrets.ErrInvalidArgument
		}
		secretName = segs[0]
		if len(segs) > 1 {
			version = segs[1]
		}
		return secretName, version, nil
	}
	segs := strings.Split(name, "/")
	secretName = segs[0]
	if len(segs) > 1 {
		version = segs[1]
	}
	return secretName, version, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var re *azcore.ResponseError
	if errors.As(err, &re) && re.StatusCode == http.StatusNotFound {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "secretnotfound") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404")
}

// Get reads the secret value (latest version when version is omitted).
func (m *Manager) Get(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	secretName, version, err := parseSecretName(name)
	if err != nil {
		return "", err
	}
	out, err := m.client.GetSecret(ctx, secretName, version, nil)
	if err != nil {
		if isNotFound(err) {
			return "", secrets.ErrNotFound
		}
		return "", pkgerrors.New(secrets.CodeUnavailable, "azure key vault get failed", err)
	}
	if out.Value == nil {
		return "", secrets.ErrNotFound
	}
	return *out.Value, nil
}

// Set creates or updates a secret value.
func (m *Manager) Set(ctx context.Context, name, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	secretName, _, err := parseSecretName(name)
	if err != nil {
		return err
	}
	_, err = m.client.SetSecret(ctx, secretName, azsecrets.SetSecretParameters{
		Value: &value,
	}, nil)
	if err != nil {
		return pkgerrors.New(secrets.CodeUnavailable, "azure key vault set failed", err)
	}
	return nil
}

// Delete soft-deletes a secret from the vault.
func (m *Manager) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	secretName, _, err := parseSecretName(name)
	if err != nil {
		return err
	}
	_, err = m.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		if isNotFound(err) {
			return secrets.ErrNotFound
		}
		return pkgerrors.New(secrets.CodeUnavailable, "azure key vault delete failed", err)
	}
	return nil
}

// Rotate replaces the secret value (generates when newValue is empty).
func (m *Manager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := m.Get(ctx, name); err != nil {
		return "", err
	}
	if newValue == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return "", secrets.ErrRotateFailed
		}
		newValue = base64.RawURLEncoding.EncodeToString(buf)
	}
	if err := m.Set(ctx, name, newValue); err != nil {
		return "", pkgerrors.New(secrets.CodeRotateFailed, "azure key vault rotate failed", err)
	}
	return newValue, nil
}
