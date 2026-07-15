// Package awssecrets implements secrets.SecretManager via AWS Secrets Manager.
//
// Get/Set(/Rotate) wrap GetSecretValue and PutSecretValue/CreateSecret.
// Inject SecretsAPI via NewFromAPI for tests.
package awssecrets

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
)

// SecretsAPI is the AWS Secrets Manager surface used by this adapter.
type SecretsAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
}

// Config configures the AWS Secrets Manager adapter.
type Config struct {
	Region          string `env:"AWS_REGION" env-default:"us-east-1"`
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`
	Endpoint        string `env:"AWS_SECRETS_ENDPOINT"`
}

// Manager implements secrets.SecretManager.
type Manager struct {
	client SecretsAPI
}

var _ secrets.SecretManager = (*Manager)(nil)

// NewFromAPI wraps an existing SecretsAPI.
func NewFromAPI(api SecretsAPI) (*Manager, error) {
	if api == nil {
		return nil, pkgerrors.New(secrets.CodeInvalidArgument, "secrets manager api is required", nil)
	}
	return &Manager{client: api}, nil
}

// New builds a Manager from AWS SDK config.
func New(ctx context.Context, cfg Config) (*Manager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, pkgerrors.New(secrets.CodeInvalidArgument, "aws region is required", nil)
	}
	opts := []func(*config.LoadOptions) error{config.WithRegion(cfg.Region)}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.New(secrets.CodeUnavailable, "failed to load aws config", err)
	}
	client := secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return NewFromAPI(client)
}

// Get reads the current secret string value.
func (m *Manager) Get(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if name == "" {
		return "", secrets.ErrInvalidArgument
	}
	out, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		var nf *smtypes.ResourceNotFoundException
		if errors.As(err, &nf) || isNotFound(err) {
			return "", secrets.ErrNotFound
		}
		return "", pkgerrors.New(secrets.CodeUnavailable, "aws secrets get failed", err)
	}
	if out == nil {
		return "", secrets.ErrNotFound
	}
	if out.SecretString != nil {
		return *out.SecretString, nil
	}
	if len(out.SecretBinary) > 0 {
		return string(out.SecretBinary), nil
	}
	return "", secrets.ErrNotFound
}

// Set creates or updates a secret string value.
func (m *Manager) Set(ctx context.Context, name, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return secrets.ErrInvalidArgument
	}
	_, err := m.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(name),
		SecretString: aws.String(value),
	})
	if err == nil {
		return nil
	}
	var nf *smtypes.ResourceNotFoundException
	if errors.As(err, &nf) || isNotFound(err) {
		_, cerr := m.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(name),
			SecretString: aws.String(value),
		})
		if cerr != nil {
			return pkgerrors.New(secrets.CodeUnavailable, "aws secrets create failed", cerr)
		}
		return nil
	}
	return pkgerrors.New(secrets.CodeUnavailable, "aws secrets put failed", err)
}

// Rotate replaces the secret (generates a value when newValue is empty).
func (m *Manager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if name == "" {
		return "", secrets.ErrInvalidArgument
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
		return "", pkgerrors.New(secrets.CodeRotateFailed, "aws secrets rotate failed", err)
	}
	return newValue, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "resourcenotfound") || strings.Contains(msg, "not found")
}
