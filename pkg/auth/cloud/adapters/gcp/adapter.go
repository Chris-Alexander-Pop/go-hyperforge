package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

const identityToolkitSignInURL = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"

type Adapter struct {
	client     *auth.Client
	apiKey     string
	httpClient *http.Client
}

// Config configures the cloud GCP identity adapter.
type Config struct {
	APIKey string
}

// New creates a GCP/Firebase cloud identity adapter.
// apiKey is required for SignIn (Identity Toolkit REST); SignUp uses Admin SDK.
func New(ctx context.Context, cfg Config) (*Adapter, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, errors.Internal("failed to initialize firebase app", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, errors.Internal("failed to create auth client", err)
	}

	return &Adapter{
		client:     client,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (a *Adapter) SignUp(ctx context.Context, email, password string, attributes map[string]string) error {
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(attributes["name"])

	_, err := a.client.CreateUser(ctx, params)
	if err != nil {
		return errors.Internal("failed to create firebase user", err)
	}
	return nil
}

func (a *Adapter) SignIn(ctx context.Context, username, password string) (*cloud.AuthResult, error) {
	if a.apiKey == "" {
		return nil, errors.InvalidArgument("APIKey required for gcp SignIn", nil)
	}
	if username == "" || password == "" {
		return nil, errors.Unauthorized("invalid credentials", nil)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"email":             username,
		"password":          password,
		"returnSecureToken": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, identityToolkitSignInURL+"?key="+a.apiKey, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Internal("failed to create sign-in request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, errors.Unavailable("identity toolkit request failed", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Internal("failed to read sign-in response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Unauthorized(fmt.Sprintf("sign-in failed with status %d", resp.StatusCode), nil)
	}

	var result struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    string `json:"expiresIn"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Internal("failed to parse sign-in response", err)
	}

	expiresIn := 0
	_, _ = fmt.Sscanf(result.ExpiresIn, "%d", &expiresIn)

	return &cloud.AuthResult{
		AccessToken:  result.IDToken,
		RefreshToken: result.RefreshToken,
		IDToken:      result.IDToken,
		ExpiresIn:    expiresIn,
	}, nil
}
