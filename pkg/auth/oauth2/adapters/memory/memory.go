package memory

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/oauth2"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type authCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	Subject             string
	Scopes              []string
	ExpiresAt           time.Time
	CodeChallenge       string
	CodeChallengeMethod string
	Used                bool
}

type issuedToken struct {
	Token     string
	Subject   string
	ClientID  string
	Scopes    []string
	ExpiresAt time.Time
	IssuedAt  time.Time
	Refresh   bool
	Revoked   bool
}

// Server is an in-memory OAuth2 AuthorizationServer + TokenIssuer.
type Server struct {
	cfg      oauth2.Config
	mu       *concurrency.SmartRWMutex
	clients  map[string]oauth2.Client
	codes    map[string]*authCode
	tokens   map[string]*issuedToken
	password oauth2.PasswordAuthenticator
	now      func() time.Time
}

// Option configures the memory server.
type Option func(*Server)

// WithPasswordAuthenticator enables the password grant.
func WithPasswordAuthenticator(a oauth2.PasswordAuthenticator) Option {
	return func(s *Server) { s.password = a }
}

// WithClock overrides the clock (tests).
func WithClock(now func() time.Time) Option {
	return func(s *Server) { s.now = now }
}

// New creates a memory authorization server.
func New(cfg oauth2.Config, opts ...Option) *Server {
	if cfg.Issuer == "" {
		cfg.Issuer = "hyperforge-oauth2"
	}
	if cfg.AccessTokenTTL == 0 {
		cfg.AccessTokenTTL = time.Hour
	}
	if cfg.RefreshTokenTTL == 0 {
		cfg.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	if cfg.AuthCodeTTL == 0 {
		cfg.AuthCodeTTL = 10 * time.Minute
	}
	s := &Server{
		cfg:     cfg,
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "oauth2-memory"}),
		clients: make(map[string]oauth2.Client),
		codes:   make(map[string]*authCode),
		tokens:  make(map[string]*issuedToken),
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Issuer implements oauth2.AuthorizationServer.
func (s *Server) Issuer() oauth2.TokenIssuer { return s }

// RegisterClient stores a client definition.
func (s *Server) RegisterClient(ctx context.Context, client oauth2.Client) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if client.ID == "" {
		return oauth2.ErrInvalidRequestMsg("client id required")
	}
	if len(client.GrantTypes) == 0 {
		client.GrantTypes = []oauth2.GrantType{oauth2.GrantAuthorizationCode, oauth2.GrantRefreshToken}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ID] = client
	return nil
}

// Authorize issues an authorization code for an authenticated subject.
func (s *Server) Authorize(ctx context.Context, req oauth2.AuthorizeRequest) (*oauth2.AuthorizeResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.ResponseType != oauth2.ResponseTypeCode && req.ResponseType != "" {
		return nil, oauth2.ErrInvalidRequestMsg("unsupported response_type")
	}
	if req.ResponseType == "" {
		req.ResponseType = oauth2.ResponseTypeCode
	}
	if req.ClientID == "" || req.Subject == "" || req.RedirectURI == "" {
		return nil, oauth2.ErrInvalidRequestMsg("client_id, subject, and redirect_uri are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[req.ClientID]
	if !ok {
		return nil, oauth2.ErrClientNotFound
	}
	if !clientAllowsGrant(client, oauth2.GrantAuthorizationCode) {
		return nil, oauth2.ErrUnsupportedGrant
	}
	if !redirectAllowed(client, req.RedirectURI) {
		return nil, oauth2.ErrInvalidRequestMsg("redirect_uri not registered")
	}

	code := randomToken("code")
	s.codes[code] = &authCode{
		Code:                code,
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Subject:             req.Subject,
		Scopes:              normalizeScopes(req.Scope, client.Scopes),
		ExpiresAt:           s.now().Add(s.cfg.AuthCodeTTL),
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}
	return &oauth2.AuthorizeResponse{
		Code:        code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
	}, nil
}

// Token handles token endpoint grants.
func (s *Server) Token(ctx context.Context, req oauth2.TokenRequest) (*oauth2.TokenResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch req.GrantType {
	case oauth2.GrantAuthorizationCode, "":
		if req.GrantType == "" {
			req.GrantType = oauth2.GrantAuthorizationCode
		}
		return s.tokenAuthCode(ctx, req)
	case oauth2.GrantClientCredentials:
		return s.tokenClientCredentials(ctx, req)
	case oauth2.GrantRefreshToken:
		return s.tokenRefresh(ctx, req)
	case oauth2.GrantPassword:
		return s.tokenPassword(ctx, req)
	default:
		return nil, oauth2.ErrUnsupportedGrant
	}
}

func (s *Server) tokenAuthCode(ctx context.Context, req oauth2.TokenRequest) (*oauth2.TokenResponse, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.authenticateClientLocked(req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !clientAllowsGrant(client, oauth2.GrantAuthorizationCode) {
		return nil, oauth2.ErrUnsupportedGrant
	}
	ac, ok := s.codes[req.Code]
	if !ok || ac.Used || s.now().After(ac.ExpiresAt) {
		return nil, oauth2.ErrInvalidGrant
	}
	if ac.ClientID != req.ClientID || ac.RedirectURI != req.RedirectURI {
		return nil, oauth2.ErrInvalidGrant
	}
	if ac.CodeChallenge != "" {
		if !verifyPKCE(ac.CodeChallenge, ac.CodeChallengeMethod, req.CodeVerifier) {
			return nil, oauth2.ErrInvalidGrant
		}
	}
	ac.Used = true
	return s.issueTokensLocked(ac.Subject, client.ID, ac.Scopes, true)
}

func (s *Server) tokenClientCredentials(ctx context.Context, req oauth2.TokenRequest) (*oauth2.TokenResponse, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.authenticateClientLocked(req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !clientAllowsGrant(client, oauth2.GrantClientCredentials) {
		return nil, oauth2.ErrUnsupportedGrant
	}
	scopes := normalizeScopes(req.Scope, client.Scopes)
	return s.issueTokensLocked(client.ID, client.ID, scopes, false)
}

func (s *Server) tokenRefresh(ctx context.Context, req oauth2.TokenRequest) (*oauth2.TokenResponse, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.authenticateClientLocked(req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !clientAllowsGrant(client, oauth2.GrantRefreshToken) {
		return nil, oauth2.ErrUnsupportedGrant
	}
	rt, ok := s.tokens[req.RefreshToken]
	if !ok || !rt.Refresh || rt.Revoked || s.now().After(rt.ExpiresAt) {
		return nil, oauth2.ErrInvalidGrant
	}
	if rt.ClientID != client.ID {
		return nil, oauth2.ErrInvalidGrant
	}
	scopes := rt.Scopes
	if len(req.Scope) > 0 {
		scopes = normalizeScopes(req.Scope, rt.Scopes)
	}
	rt.Revoked = true
	return s.issueTokensLocked(rt.Subject, client.ID, scopes, true)
}

func (s *Server) tokenPassword(ctx context.Context, req oauth2.TokenRequest) (*oauth2.TokenResponse, error) {
	if s.password == nil {
		return nil, oauth2.ErrUnsupportedGrant
	}
	subject, err := s.password.Authenticate(ctx, req.Username, req.Password)
	if err != nil {
		return nil, errors.Unauthorized("invalid credentials", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.authenticateClientLocked(req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !clientAllowsGrant(client, oauth2.GrantPassword) {
		return nil, oauth2.ErrUnsupportedGrant
	}
	scopes := normalizeScopes(req.Scope, client.Scopes)
	return s.issueTokensLocked(subject, client.ID, scopes, true)
}

// IssueAccessToken implements oauth2.TokenIssuer.
func (s *Server) IssueAccessToken(ctx context.Context, subject, clientID string, scopes []string) (*oauth2.TokenResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.issueTokensLocked(subject, clientID, scopes, true)
}

// Revoke implements oauth2.TokenIssuer.
func (s *Server) Revoke(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tokens[token]; ok {
		t.Revoked = true
	}
	return nil
}

// Introspect implements oauth2.TokenIssuer.
func (s *Server) Introspect(ctx context.Context, token string) (*oauth2.TokenClaims, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tokens[token]
	if !ok || t.Revoked || t.Refresh || s.now().After(t.ExpiresAt) {
		return nil, errors.Unauthorized("invalid token", nil)
	}
	return &oauth2.TokenClaims{
		Subject:   t.Subject,
		ClientID:  t.ClientID,
		Scopes:    append([]string(nil), t.Scopes...),
		Issuer:    s.cfg.Issuer,
		ExpiresAt: t.ExpiresAt,
		IssuedAt:  t.IssuedAt,
	}, nil
}

func (s *Server) authenticateClientLocked(id, secret string) (oauth2.Client, error) {
	client, ok := s.clients[id]
	if !ok {
		return oauth2.Client{}, oauth2.ErrInvalidClient
	}
	if client.Secret != "" && subtle.ConstantTimeCompare([]byte(client.Secret), []byte(secret)) != 1 {
		return oauth2.Client{}, oauth2.ErrInvalidClient
	}
	return client, nil
}

func (s *Server) issueTokensLocked(subject, clientID string, scopes []string, withRefresh bool) (*oauth2.TokenResponse, error) {
	now := s.now()
	access := randomToken("atk")
	s.tokens[access] = &issuedToken{
		Token:     access,
		Subject:   subject,
		ClientID:  clientID,
		Scopes:    scopes,
		ExpiresAt: now.Add(s.cfg.AccessTokenTTL),
		IssuedAt:  now,
	}
	resp := &oauth2.TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.cfg.AccessTokenTTL.Seconds()),
		Scope:       strings.Join(scopes, " "),
	}
	if withRefresh {
		refresh := randomToken("rtk")
		s.tokens[refresh] = &issuedToken{
			Token:     refresh,
			Subject:   subject,
			ClientID:  clientID,
			Scopes:    scopes,
			ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
			IssuedAt:  now,
			Refresh:   true,
		}
		resp.RefreshToken = refresh
	}
	return resp, nil
}

func clientAllowsGrant(c oauth2.Client, g oauth2.GrantType) bool {
	for _, gt := range c.GrantTypes {
		if gt == g {
			return true
		}
	}
	return false
}

func redirectAllowed(c oauth2.Client, uri string) bool {
	if len(c.RedirectURIs) == 0 {
		return true
	}
	for _, r := range c.RedirectURIs {
		if r == uri {
			return true
		}
	}
	return false
}

func normalizeScopes(requested, allowed []string) []string {
	if len(requested) == 0 {
		return append([]string(nil), allowed...)
	}
	if len(allowed) == 0 {
		return append([]string(nil), requested...)
	}
	allow := make(map[string]struct{}, len(allowed))
	for _, s := range allowed {
		allow[s] = struct{}{}
	}
	var out []string
	for _, s := range requested {
		if _, ok := allow[s]; ok {
			out = append(out, s)
		}
	}
	return out
}

func verifyPKCE(challenge, method, verifier string) bool {
	if verifier == "" {
		return false
	}
	switch strings.ToLower(method) {
	case "", "plain":
		return subtle.ConstantTimeCompare([]byte(challenge), []byte(verifier)) == 1
	case "s256":
		sum := sha256.Sum256([]byte(verifier))
		encoded := base64.RawURLEncoding.EncodeToString(sum[:])
		return subtle.ConstantTimeCompare([]byte(challenge), []byte(encoded)) == 1
	default:
		return false
	}
}

func randomToken(prefix string) string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to time-based uniqueness.
		return prefix + "_" + hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return prefix + "_" + hex.EncodeToString(b)
}
