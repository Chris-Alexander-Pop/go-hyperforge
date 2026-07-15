package saml

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Domain helpers for the SAML client skeleton.
var (
	// ErrInvalidResponse is returned when SAMLResponse cannot be parsed.
	ErrInvalidResponse = errors.InvalidArgument("invalid saml response", nil)

	// ErrInvalidConfig is returned when SP/IdP configuration is incomplete.
	ErrInvalidConfig = errors.InvalidArgument("invalid saml configuration", nil)

	// ErrUnimplementedSSO marks full XML crypto / production SSO as not available.
	ErrUnimplementedSSO = errors.Unimplemented("saml xml signature validation not implemented", nil)
)
