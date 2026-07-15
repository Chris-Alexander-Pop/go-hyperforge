// Package memory provides an in-memory SAML SP client for tests.
//
// ParseResponse accepts a base64-encoded JSON test double:
//
//	{"sub":"user-1","email":"a@b.c","roles":["admin"]}
//
// It does not parse real SAML XML. ValidateXMLSignature always returns
// Unimplemented.
package memory

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/saml"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Client is an in-memory SAML SP stub.
type Client struct {
	cfg saml.Config
	mu  *concurrency.SmartRWMutex
	// subjects maps RelayState → pre-registered claims (optional).
	subjects map[string]*auth.Claims
}

// New creates a memory SAML client. EntityID, ACSURL, and IdPSSOURL are required.
func New(cfg saml.Config) (*Client, error) {
	if strings.TrimSpace(cfg.EntityID) == "" ||
		strings.TrimSpace(cfg.ACSURL) == "" ||
		strings.TrimSpace(cfg.IdPSSOURL) == "" {
		return nil, saml.ErrInvalidConfig
	}
	return &Client{
		cfg:      cfg,
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "saml-memory"}),
		subjects: make(map[string]*auth.Claims),
	}, nil
}

// RegisterRelayState associates RelayState with claims for ParseResponse shortcuts.
func (c *Client) RegisterRelayState(relayState string, claims *auth.Claims) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subjects[relayState] = claims
}

// MetadataXML returns a minimal SP metadata stub (not a full SAML metadata doc).
func (c *Client) MetadataXML(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	xml := fmt.Sprintf(
		`<EntityDescriptor entityID=%q><SPSSODescriptor><AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location=%q/></SPSSODescriptor></EntityDescriptor>`,
		c.cfg.EntityID, c.cfg.ACSURL,
	)
	return []byte(xml), nil
}

// AuthnRequestURL builds a redirect URL with SAMLRequest and RelayState query params.
// SAMLRequest is a stub identifier (not a real Deflate+base64 AuthnRequest).
func (c *Client) AuthnRequestURL(ctx context.Context, relayState string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	u, err := url.Parse(c.cfg.IdPSSOURL)
	if err != nil {
		return "", saml.ErrInvalidConfig
	}
	q := u.Query()
	q.Set("SAMLRequest", base64.StdEncoding.EncodeToString([]byte("AuthnRequest:"+c.cfg.EntityID)))
	if relayState != "" {
		q.Set("RelayState", relayState)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type stubAssertion struct {
	Subject  string   `json:"sub"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Issuer   string   `json:"iss"`
	Audience string   `json:"aud"`
}

// ParseResponse decodes a base64 JSON test double into auth.Claims.
// If RelayState was registered, those claims win when SAMLResponse is empty.
func (c *Client) ParseResponse(ctx context.Context, req saml.AssertionConsumerRequest) (*auth.Claims, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.SAMLResponse) == "" {
		c.mu.RLock()
		claims, ok := c.subjects[req.RelayState]
		c.mu.RUnlock()
		if ok && claims != nil {
			cp := *claims
			return &cp, nil
		}
		return nil, saml.ErrInvalidResponse
	}

	raw, err := base64.StdEncoding.DecodeString(req.SAMLResponse)
	if err != nil {
		// try raw URL-safe
		raw, err = base64.URLEncoding.DecodeString(req.SAMLResponse)
		if err != nil {
			return nil, saml.ErrInvalidResponse
		}
	}

	var stub stubAssertion
	if err := json.Unmarshal(raw, &stub); err != nil {
		return nil, saml.ErrInvalidResponse
	}
	if stub.Subject == "" {
		return nil, saml.ErrInvalidResponse
	}

	iss := stub.Issuer
	if iss == "" {
		iss = c.cfg.IdPEntityID
	}
	aud := stub.Audience
	if aud == "" {
		aud = c.cfg.EntityID
	}

	return &auth.Claims{
		Subject:  stub.Subject,
		Issuer:   iss,
		Audience: []string{aud},
		Email:    stub.Email,
		Roles:    stub.Roles,
	}, nil
}

// ValidateXMLSignature always returns Unimplemented — full SSO crypto is out of scope.
func (c *Client) ValidateXMLSignature(ctx context.Context, rawXML []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = rawXML
	return saml.ErrUnimplementedSSO
}

var _ saml.Client = (*Client)(nil)
