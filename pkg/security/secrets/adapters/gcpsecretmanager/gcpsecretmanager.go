// Package gcpsecretmanager implements secrets.SecretManager via GCP Secret Manager.
//
// Get/Set wrap AccessSecretVersion and CreateSecret/AddSecretVersion.
// Inject SecretsAPI via NewFromAPI for tests.
package gcpsecretmanager

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SecretsAPI is the GCP Secret Manager surface used by this adapter.
type SecretsAPI interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	Close() error
}

// Config configures the GCP Secret Manager adapter.
type Config struct {
	// ProjectID is the GCP project (required for relative secret names).
	ProjectID string `env:"GCP_PROJECT" env-default:""`

	// CredentialsFile is an optional path to a service account JSON key.
	CredentialsFile string `env:"GOOGLE_APPLICATION_CREDENTIALS"`

	// Endpoint overrides the API endpoint (tests).
	Endpoint string `env:"GCP_SECRETMANAGER_ENDPOINT"`
}

// Manager implements secrets.SecretManager.
type Manager struct {
	client    SecretsAPI
	projectID string
	owns      bool
}

var _ secrets.SecretManager = (*Manager)(nil)

// NewFromAPI wraps an existing SecretsAPI.
// projectID is used when name is not a full resource path.
func NewFromAPI(api SecretsAPI, projectID string) (*Manager, error) {
	if api == nil {
		return nil, errors.New(secrets.CodeInvalidArgument, "secret manager api is required", nil)
	}
	return &Manager{client: api, projectID: projectID}, nil
}

// New builds a Manager from GCP client options.
func New(ctx context.Context, cfg Config) (*Manager, error) {
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
	client, err := secretmanager.NewClient(ctx, opts...)
	if err != nil {
		return nil, errors.New(secrets.CodeUnavailable, "failed to create gcp secretmanager client", err)
	}
	m, err := NewFromAPI(client, cfg.ProjectID)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	m.owns = true
	return m, nil
}

// Close releases the underlying client when owned by New.
func (m *Manager) Close() error {
	if m.owns && m.client != nil {
		return m.client.Close()
	}
	return nil
}

func (m *Manager) secretName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", secrets.ErrInvalidArgument
	}
	if strings.HasPrefix(name, "projects/") {
		return name, nil
	}
	if m.projectID == "" {
		return "", errors.New(secrets.CodeInvalidArgument, "gcp project id is required for relative secret names", nil)
	}
	return fmt.Sprintf("projects/%s/secrets/%s", m.projectID, name), nil
}

// Get reads the latest secret version payload.
func (m *Manager) Get(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	parent, err := m.secretName(name)
	if err != nil {
		return "", err
	}
	out, err := m.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: parent + "/versions/latest",
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", secrets.ErrNotFound
		}
		return "", errors.New(secrets.CodeUnavailable, "gcp secretmanager get failed", err)
	}
	if out == nil || out.Payload == nil {
		return "", secrets.ErrNotFound
	}
	return string(out.Payload.Data), nil
}

// Set creates the secret if missing and adds a new version.
func (m *Manager) Set(ctx context.Context, name, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parent, err := m.secretName(name)
	if err != nil {
		return err
	}

	// Try add version first (secret exists).
	_, err = m.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  parent,
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	})
	if err == nil {
		return nil
	}
	if status.Code(err) != codes.NotFound {
		return errors.New(secrets.CodeUnavailable, "gcp secretmanager add version failed", err)
	}

	// Create secret then add version.
	parts := strings.Split(parent, "/")
	if len(parts) < 4 {
		return secrets.ErrInvalidArgument
	}
	project := strings.Join(parts[:2], "/") // projects/{id}
	secretID := parts[len(parts)-1]
	_, err = m.client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   project,
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return errors.New(secrets.CodeUnavailable, "gcp secretmanager create failed", err)
	}
	_, err = m.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  parent,
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	})
	if err != nil {
		return errors.New(secrets.CodeUnavailable, "gcp secretmanager add version failed", err)
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
		return "", errors.New(secrets.CodeRotateFailed, "gcp secretmanager rotate failed", err)
	}
	return newValue, nil
}
