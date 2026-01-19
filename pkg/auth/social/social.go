package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type ProviderType string

const (
	ProviderGoogle   ProviderType = "google"
	ProviderGitHub   ProviderType = "github"
	ProviderFacebook ProviderType = "facebook"
	// Apple requires special handling typically (JWT client secret), omitting for brevity unless requested specifically
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

	return &GenericOAuth2{config: conf, userInfoEndpoint: userInfoURL}, nil
}

func (p *GenericOAuth2) GetLoginURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.config.AuthCodeURL(state, opts...)
}

func (p *GenericOAuth2) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "oauth exchange failed")
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

	// Parsing varies slightly by provider, but most follow basic JSON.
	// For robust production, use specific structs per provider. We use a generic map helper.
	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// Normalize
	u := &UserInfo{}

	// ID
	if id, ok := raw["id"].(string); ok {
		u.ID = id
	} // Google, GitHub, FB
	if id, ok := raw["id"].(float64); ok {
		u.ID = fmt.Sprintf("%.0f", id)
	} // GitHub sometimes?

	// Email
	if email, ok := raw["email"].(string); ok {
		u.Email = email
	}

	// Name
	if name, ok := raw["name"].(string); ok {
		u.Name = name
	}

	return u, nil
}
