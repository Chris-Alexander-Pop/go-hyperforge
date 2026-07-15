package oauth2

import (
	"context"
	"time"
)

// GrantType identifies an OAuth 2.0 grant.
type GrantType string

const (
	GrantAuthorizationCode GrantType = "authorization_code"
	GrantClientCredentials GrantType = "client_credentials"
	GrantRefreshToken      GrantType = "refresh_token"
	GrantPassword          GrantType = "password" // resource-owner; discouraged, optional
)

// ResponseType for the authorize endpoint.
type ResponseType string

const (
	ResponseTypeCode ResponseType = "code"
)

// Client is a registered OAuth2 client application.
type Client struct {
	ID           string
	Secret       string
	RedirectURIs []string
	GrantTypes   []GrantType
	Scopes       []string
}

// Config configures an authorization server.
type Config struct {
	// Issuer is the token issuer claim / identifier (e.g. https://auth.example.com).
	Issuer string `env:"AUTH_OAUTH2_ISSUER" env-default:"hyperforge-oauth2"`

	// AccessTokenTTL is how long access tokens remain valid.
	AccessTokenTTL time.Duration `env:"AUTH_OAUTH2_ACCESS_TTL" env-default:"1h"`

	// RefreshTokenTTL is how long refresh tokens remain valid.
	RefreshTokenTTL time.Duration `env:"AUTH_OAUTH2_REFRESH_TTL" env-default:"720h"`

	// AuthCodeTTL is how long authorization codes remain redeemable.
	AuthCodeTTL time.Duration `env:"AUTH_OAUTH2_CODE_TTL" env-default:"10m"`
}

// AuthorizeRequest is the shape of an /authorize endpoint request
// after the resource owner has already authenticated (subject is set).
type AuthorizeRequest struct {
	ResponseType ResponseType
	ClientID     string
	RedirectURI  string
	Scope        []string
	State        string
	// Subject is the authenticated resource-owner user ID.
	Subject string
	// PKCE (optional)
	CodeChallenge       string
	CodeChallengeMethod string // "S256" or "plain"
}

// AuthorizeResponse is the successful /authorize result (redirect parameters).
type AuthorizeResponse struct {
	Code        string
	State       string
	RedirectURI string
}

// TokenRequest is the shape of an /token endpoint request.
type TokenRequest struct {
	GrantType    GrantType
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	RefreshToken string
	Scope        []string
	Username     string
	Password     string
	// PKCE verifier (authorization_code)
	CodeVerifier string
}

// TokenResponse is a successful /token response body.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"` // opaque optional; not a full OIDC ID token
}

// TokenClaims are the introspected claims for an issued access token.
type TokenClaims struct {
	Subject   string
	ClientID  string
	Scopes    []string
	Issuer    string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// TokenIssuer issues and revokes access/refresh tokens.
type TokenIssuer interface {
	IssueAccessToken(ctx context.Context, subject, clientID string, scopes []string) (*TokenResponse, error)
	Revoke(ctx context.Context, token string) error
	Introspect(ctx context.Context, token string) (*TokenClaims, error)
}

// AuthorizationServer is the authorize + token endpoint surface.
type AuthorizationServer interface {
	// RegisterClient stores a client (memory adapter) or returns ErrInvalidConfig.
	RegisterClient(ctx context.Context, client Client) error

	// Authorize handles the authorization-code flow (post-authentication).
	Authorize(ctx context.Context, req AuthorizeRequest) (*AuthorizeResponse, error)

	// Token handles the token endpoint for supported grants.
	Token(ctx context.Context, req TokenRequest) (*TokenResponse, error)

	// Issuer exposes the TokenIssuer used by this server.
	Issuer() TokenIssuer
}

// PasswordAuthenticator validates resource-owner credentials for the password grant.
// Optional; memory adapter accepts a hook or rejects password grants when unset.
type PasswordAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (subject string, err error)
}
