// Package saml provides a thin SAML 2.0 Service Provider client skeleton.
//
// This is enough to wire an assertion-consumer shape and IdP redirect URL.
// Full XML signature / encryption validation and production SSO are not
// implemented — ParseResponse on adapters may accept test doubles only, and
// ValidateXMLSignature returns Unimplemented.
package saml

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
)

// Config configures a SAML Service Provider client.
type Config struct {
	// EntityID is the SP entity ID (often the metadata URL).
	EntityID string `env:"AUTH_SAML_ENTITY_ID"`

	// ACSURL is the Assertion Consumer Service URL (POST binding).
	ACSURL string `env:"AUTH_SAML_ACS_URL"`

	// IdPSSOURL is the IdP single sign-on redirect URL.
	IdPSSOURL string `env:"AUTH_SAML_IDP_SSO_URL"`

	// IdPEntityID is the expected IdP entity ID (optional for stubs).
	IdPEntityID string `env:"AUTH_SAML_IDP_ENTITY_ID"`
}

// AssertionConsumerRequest is the shape of an ACS POST body.
type AssertionConsumerRequest struct {
	// SAMLResponse is the base64-encoded SAML Response (or a test-double payload).
	SAMLResponse string
	// RelayState is the optional opaque state echoed from AuthnRequestURL.
	RelayState string
}

// Client is a SAML 2.0 SP client.
type Client interface {
	// MetadataXML returns SP metadata XML bytes (may be a minimal stub).
	MetadataXML(ctx context.Context) ([]byte, error)

	// AuthnRequestURL builds a redirect URL to the IdP SSO endpoint.
	AuthnRequestURL(ctx context.Context, relayState string) (string, error)

	// ParseResponse consumes an ACS request and returns identity claims.
	// Adapters may accept simplified test payloads; production XML crypto is separate.
	ParseResponse(ctx context.Context, req AssertionConsumerRequest) (*auth.Claims, error)

	// ValidateXMLSignature validates a raw SAML Response XML document.
	// Full SSO crypto is not part of this skeleton — expect Unimplemented.
	ValidateXMLSignature(ctx context.Context, rawXML []byte) error
}
