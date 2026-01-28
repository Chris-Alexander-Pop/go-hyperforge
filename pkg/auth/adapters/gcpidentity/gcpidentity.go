package gcpidentity

import (
	"context"
	"fmt"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	pkgauth "github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Config configures the GCP Identity (Firebase Auth) adapter.
type Config struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"AUTH_GCP_PROJECT_ID" validate:"required"`

	// CredentialsFile is the path to the service account key file (optional).
	CredentialsFile string `env:"AUTH_GCP_CREDENTIALS_FILE"`

	// APIKey is the Firebase Web API Key (required for username/password login).
	// This is not typically in the service account.
	APIKey string `env:"AUTH_GCP_API_KEY"`
}

// Adapter implements auth.IdentityProvider and auth.Verifier for GCP/Firebase.
type Adapter struct {
	authClient *auth.Client
	apiKey     string
}

// New creates a new GCP Identity adapter.
func New(ctx context.Context, cfg Config) (*Adapter, error) {
	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, errors.Internal("failed to read credentials file", err)
		}
		// Use default scopes for Firebase
		creds, err := google.CredentialsFromJSON(ctx, b, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, errors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, opts...)
	if err != nil {
		return nil, errors.Internal("failed to initialize firebase app", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, errors.Internal("failed to create auth client", err)
	}

	return &Adapter{
		authClient: client,
		apiKey:     cfg.APIKey,
	}, nil
}

// Login authenticates a user with username (email) and password.
// Note: Firebase Admin SDK does NOT support sign-in with password.
// We must use the Firebase Auth REST API for this.
func (a *Adapter) Login(ctx context.Context, username, password string) (*pkgauth.Claims, error) {
	if a.apiKey == "" {
		return nil, errors.InvalidArgument("api key required for gcp login", nil)
	}

	// In a complete implementation, this would call the Identity Toolkit API
	// https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=[API_KEY]
	// Since we avoid making manual HTTP calls if possible and strictly use SDKs where available,
	// checking if our SDK imports support this. They generally don't for security reasons (client vs admin).

	// For compliance with the interface, we'll mark this as not implemented or Stub it
	// if the user provided API key is set, we could technically do it.

	return nil, errors.Unimplemented("gcp/firebase password login requires client sdk or rest api call", nil)
}

// Verify validates a Firebase ID token.
func (a *Adapter) Verify(ctx context.Context, token string) (*pkgauth.Claims, error) {
	t, err := a.authClient.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, errors.Unauthorized("invalid token", err)
	}

	// Extract standard claims
	claims := &pkgauth.Claims{
		Subject:   t.UID,
		Issuer:    t.Issuer,
		Audience:  []string{t.Audience},
		ExpiresAt: t.Expires,
		IssuedAt:  t.IssuedAt,
		Metadata:  t.Claims,
	}

	// Try to get email if available
	if email, ok := t.Claims["email"].(string); ok {
		claims.Email = email
	}

	// Try to get role from claims or custom claims
	if role, ok := t.Claims["role"].(string); ok {
		claims.Role = role
	} else if roles, ok := t.Claims["roles"].([]interface{}); ok {
		// Just take the first one or join them
		var s []string
		for _, r := range roles {
			s = append(s, fmt.Sprintf("%v", r))
		}
		claims.Role = strings.Join(s, ",")
	}

	return claims, nil
}
