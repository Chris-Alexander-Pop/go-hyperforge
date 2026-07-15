package gcpidentity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	pkgauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const identityToolkitSignInURL = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"

// Config configures the GCP Identity (Firebase Auth) adapter.
type Config struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"AUTH_GCP_PROJECT_ID" validate:"required"`

	// CredentialsFile is the path to the service account key file (optional).
	CredentialsFile string `env:"AUTH_GCP_CREDENTIALS_FILE"`

	// APIKey is the Firebase Web API Key (required for username/password login).
	APIKey string `env:"AUTH_GCP_API_KEY"`
}

// Adapter implements auth.IdentityProvider and auth.Verifier for GCP/Firebase.
type Adapter struct {
	authClient *auth.Client
	apiKey     string
	httpClient *http.Client
}

// Option configures the adapter.
type Option func(*Adapter)

// WithHTTPClient overrides the HTTP client used for Identity Toolkit login.
func WithHTTPClient(c *http.Client) Option {
	return func(a *Adapter) {
		if c != nil {
			a.httpClient = c
		}
	}
}

// New creates a new GCP Identity adapter.
func New(ctx context.Context, cfg Config, opts ...Option) (*Adapter, error) {
	if cfg.ProjectID == "" {
		return nil, pkgauth.ErrInvalidConfigMsg("ProjectID is required", nil)
	}

	var clientOpts []option.ClientOption
	if cfg.CredentialsFile != "" {
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, errors.Internal("failed to read credentials file", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, b, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, errors.Internal("failed to parse credentials", err)
		}
		clientOpts = append(clientOpts, option.WithCredentials(creds))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, clientOpts...)
	if err != nil {
		return nil, errors.Internal("failed to initialize firebase app", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, errors.Internal("failed to create auth client", err)
	}

	a := &Adapter{
		authClient: client,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a, nil
}

// Login authenticates via the Identity Toolkit REST API (signInWithPassword).
func (a *Adapter) Login(ctx context.Context, username, password string) (*pkgauth.Claims, error) {
	if a.apiKey == "" {
		return nil, pkgauth.ErrInvalidConfigMsg("APIKey required for gcp password login", nil)
	}
	if username == "" || password == "" {
		return nil, pkgauth.ErrInvalidCredentials
	}

	body, _ := json.Marshal(map[string]interface{}{
		"email":             username,
		"password":          password,
		"returnSecureToken": true,
	})
	url := identityToolkitSignInURL + "?key=" + a.apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Internal("failed to create login request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, errors.Unavailable("gcp identity toolkit request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Internal("failed to read login response", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(respBody, &apiErr)
		msg := apiErr.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("login failed with status %d", resp.StatusCode)
		}
		return nil, errors.Unauthorized(msg, nil)
	}

	var result struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    string `json:"expiresIn"`
		LocalID      string `json:"localId"`
		Email        string `json:"email"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Internal("failed to parse login response", err)
	}

	if result.IDToken != "" {
		claims, verr := a.Verify(ctx, result.IDToken)
		if verr == nil {
			if claims.Metadata == nil {
				claims.Metadata = map[string]interface{}{}
			}
			claims.Metadata["refresh_token"] = result.RefreshToken
			claims.Metadata["expires_in"] = result.ExpiresIn
			return claims, nil
		}
	}

	return &pkgauth.Claims{
		Subject: result.LocalID,
		Issuer:  "https://securetoken.google.com/",
		Email:   result.Email,
		Metadata: map[string]interface{}{
			"id_token":      result.IDToken,
			"refresh_token": result.RefreshToken,
			"expires_in":    result.ExpiresIn,
		},
	}, nil
}

// Verify validates a Firebase ID token via the Admin SDK.
func (a *Adapter) Verify(ctx context.Context, token string) (*pkgauth.Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, pkgauth.ErrInvalidToken
	}

	t, err := a.authClient.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, pkgauth.ErrInvalidTokenWrap(err)
	}

	claims := &pkgauth.Claims{
		Subject:   t.UID,
		Issuer:    t.Issuer,
		Audience:  []string{t.Audience},
		ExpiresAt: t.Expires,
		IssuedAt:  t.IssuedAt,
		Metadata:  t.Claims,
	}

	if email, ok := t.Claims["email"].(string); ok {
		claims.Email = email
	}
	if role, ok := t.Claims["role"].(string); ok {
		claims.Roles = append(claims.Roles, role)
	}
	if roles, ok := t.Claims["roles"].([]interface{}); ok {
		for _, r := range roles {
			claims.Roles = append(claims.Roles, fmt.Sprintf("%v", r))
		}
	}

	return claims, nil
}

var (
	_ pkgauth.IdentityProvider = (*Adapter)(nil)
	_ pkgauth.Verifier         = (*Adapter)(nil)
)
