package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type ProviderType string

const (
	ProviderGoogle   ProviderType = "google"
	ProviderGitHub   ProviderType = "github"
	ProviderFacebook ProviderType = "facebook"
	ProviderApple    ProviderType = "apple"
)

// UserInfo normalized from providers
type UserInfo struct {
	ID    string
	Email string
	Name  string
}

// Provider defines the flow
type Provider interface {
	GetLoginURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string) (*UserInfo, error)
}

type GenericOAuth2 struct {
	config           *oauth2.Config
	userInfoEndpoint string
	providerType     ProviderType
}

func New(t ProviderType, clientID, clientSecret, redirectURL string) (Provider, error) {
	var endpoint oauth2.Endpoint
	var userInfoURL string
	var scopes []string

	switch t {
	case ProviderGoogle:
		endpoint = google.Endpoint
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
		scopes = []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}
	case ProviderGitHub:
		endpoint = github.Endpoint
		userInfoURL = "https://api.github.com/user"
		scopes = []string{"user:email"}
	case ProviderFacebook:
		endpoint = facebook.Endpoint
		userInfoURL = "https://graph.facebook.com/me?fields=id,name,email"
		scopes = []string{"email"}
	case ProviderApple:
		// Sign in with Apple — identity comes from id_token, not a userinfo URL.
		// clientSecret must be the Apple-signed JWT client secret.
		endpoint = endpoints.Apple
		scopes = []string{"name", "email"}
	default:
		return nil, errors.InvalidArgument(fmt.Sprintf("unsupported provider: %s", t), nil)
	}

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	return &GenericOAuth2{config: conf, userInfoEndpoint: userInfoURL, providerType: t}, nil
}

func (p *GenericOAuth2) GetLoginURL(state string, opts ...oauth2.AuthCodeOption) string {
	if p.providerType == ProviderApple {
		// Apple requires response_mode=form_post when requesting name/email.
		opts = append([]oauth2.AuthCodeOption{
			oauth2.SetAuthURLParam("response_mode", "form_post"),
			oauth2.SetAuthURLParam("response_type", "code id_token"),
		}, opts...)
	}
	return p.config.AuthCodeURL(state, opts...)
}

func (p *GenericOAuth2) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "oauth exchange failed")
	}

	if p.providerType == ProviderApple {
		return userInfoFromAppleIDToken(token)
	}

	client := p.config.Client(ctx, token)
	resp, err := client.Get(p.userInfoEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user info")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Internal(fmt.Sprintf("provider returned status: %d", resp.StatusCode), nil)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	u := &UserInfo{}

	if id, ok := raw["id"].(string); ok {
		u.ID = id
	}
	if id, ok := raw["id"].(float64); ok {
		u.ID = fmt.Sprintf("%.0f", id)
	}
	if email, ok := raw["email"].(string); ok {
		u.Email = email
	}
	if name, ok := raw["name"].(string); ok {
		u.Name = name
	}

	return u, nil
}

// UserInfoFromIDToken parses an Apple (or similar) id_token payload without
// verifying the signature. Useful for tests and for extracting claims after
// a successful OAuth token exchange. Production callers should also verify
// the JWT against Apple's JWKS when required.
func UserInfoFromIDToken(idToken string) (*UserInfo, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return nil, errors.InvalidArgument("malformed id_token", nil)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, errors.InvalidArgument("failed to decode id_token payload", err)
		}
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.InvalidArgument("failed to parse id_token claims", err)
	}
	if claims.Sub == "" {
		return nil, errors.InvalidArgument("id_token missing sub", nil)
	}

	return &UserInfo{
		ID:    claims.Sub,
		Email: claims.Email,
		Name:  claims.Name,
	}, nil
}

// userInfoFromAppleIDToken extracts sub/email from Apple's id_token.
func userInfoFromAppleIDToken(token *oauth2.Token) (*UserInfo, error) {
	raw, _ := token.Extra("id_token").(string)
	if raw == "" {
		return nil, errors.Internal("apple token response missing id_token", nil)
	}
	return UserInfoFromIDToken(raw)
}
