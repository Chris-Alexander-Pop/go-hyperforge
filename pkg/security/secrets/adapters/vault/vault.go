package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

const defaultMount = "secret"
const secretField = "value"

// Config configures the Vault KV v2 client.
type Config struct {
	// Address is the Vault base URL (e.g. https://vault.example.com:8200).
	Address string `env:"VAULT_ADDR" validate:"required"`

	// Token is the Vault auth token (X-Vault-Token).
	Token string `env:"VAULT_TOKEN" validate:"required"`

	// Mount is the KV v2 secrets engine mount path (default "secret").
	Mount string `env:"VAULT_KV_MOUNT" env-default:"secret"`

	// Namespace is an optional Vault Enterprise namespace (X-Vault-Namespace).
	Namespace string `env:"VAULT_NAMESPACE"`

	// HTTPClient is optional; defaults to a 15s timeout client.
	HTTPClient *http.Client

	// Retrier wraps Vault HTTP calls; nil uses resilience.DefaultRetryConfig.
	Retrier resilience.Retrier
}

// Validate checks required Config fields via pkg/validator.
func (c Config) Validate() error {
	if err := validator.New().ValidateStruct(context.Background(), c); err != nil {
		if errors.IsCode(err, errors.CodeInvalidArgument) {
			return err
		}
		return errors.New(secrets.CodeInvalidArgument, "invalid vault config", err)
	}
	return nil
}

// Manager is a Vault KV v2 SecretManager.
type Manager struct {
	addr      string
	token     string
	mount     string
	namespace string
	client    *http.Client
	retrier   resilience.Retrier
}

// Ensure Manager implements secrets.SecretManager.
var _ secrets.SecretManager = (*Manager)(nil)

// New creates a Vault KV v2 secret manager.
func New(cfg Config) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	mount := cfg.Mount
	if mount == "" {
		mount = defaultMount
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	retrier := cfg.Retrier
	if retrier == nil {
		retrier = resilience.NewRetrier(resilience.DefaultRetryConfig())
	}
	return &Manager{
		addr:      strings.TrimRight(cfg.Address, "/"),
		token:     cfg.Token,
		mount:     strings.Trim(mount, "/"),
		namespace: cfg.Namespace,
		client:    client,
		retrier:   retrier,
	}, nil
}

type kvReadResponse struct {
	Data struct {
		Data map[string]interface{} `json:"data"`
	} `json:"data"`
}

type kvWriteBody struct {
	Data map[string]interface{} `json:"data"`
}

func (m *Manager) dataURL(name string) string {
	name = strings.Trim(name, "/")
	return fmt.Sprintf("%s/v1/%s/data/%s", m.addr, m.mount, name)
}

func (m *Manager) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	var resp *http.Response
	err := m.retrier.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return errors.New(secrets.CodeInvalidArgument, "failed to build vault request", err)
		}
		req.Header.Set("X-Vault-Token", m.token)
		if m.namespace != "" {
			req.Header.Set("X-Vault-Namespace", m.namespace)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		r, err := m.client.Do(req)
		if err != nil {
			return errors.New(secrets.CodeUnavailable, "vault unreachable", err)
		}
		resp = r
		return nil
	})
	return resp, err
}

// Get reads the latest KV v2 secret value for name (field "value").
func (m *Manager) Get(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if name == "" {
		return "", secrets.ErrInvalidArgument
	}

	resp, err := m.do(ctx, http.MethodGet, m.dataURL(name), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", errors.New(secrets.CodeUnavailable, "failed to read vault response", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var out kvReadResponse
		if err := json.Unmarshal(body, &out); err != nil {
			return "", errors.New(secrets.CodeUnavailable, "invalid vault response", err)
		}
		if out.Data.Data == nil {
			return "", secrets.ErrNotFound
		}
		v, ok := out.Data.Data[secretField]
		if !ok {
			return "", secrets.ErrNotFound
		}
		s, ok := v.(string)
		if !ok {
			return "", errors.New(secrets.CodeInvalidArgument, "vault secret value is not a string", nil)
		}
		return s, nil
	case http.StatusNotFound:
		return "", secrets.ErrNotFound
	case http.StatusForbidden, http.StatusUnauthorized:
		return "", errors.New(secrets.CodeUnavailable, "vault auth failed", nil)
	default:
		return "", errors.New(secrets.CodeUnavailable, fmt.Sprintf("vault get returned %d", resp.StatusCode), nil)
	}
}

// Set writes a KV v2 secret (creates a new version).
func (m *Manager) Set(ctx context.Context, name, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return secrets.ErrInvalidArgument
	}

	payload, err := json.Marshal(kvWriteBody{Data: map[string]interface{}{secretField: value}})
	if err != nil {
		return errors.New(secrets.CodeInvalidArgument, "failed to marshal vault payload", err)
	}

	resp, err := m.do(ctx, http.MethodPost, m.dataURL(name), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusCreated:
		return nil
	case http.StatusForbidden, http.StatusUnauthorized:
		return errors.New(secrets.CodeUnavailable, "vault auth failed", nil)
	default:
		return errors.New(secrets.CodeUnavailable, fmt.Sprintf("vault set returned %d", resp.StatusCode), nil)
	}
}

// Rotate replaces the secret with newValue (or a generated value when empty).
func (m *Manager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if name == "" {
		return "", secrets.ErrInvalidArgument
	}

	// Ensure the secret exists before rotating.
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
		return "", errors.New(secrets.CodeRotateFailed, "vault rotate write failed", err)
	}
	return newValue, nil
}
