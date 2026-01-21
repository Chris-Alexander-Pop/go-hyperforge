package webauthn

import (
	"context"
)

// Config configures the WebAuthn service.
type Config struct {
	// Driver specifies the WebAuthn storage driver.
	Driver string `env:"AUTH_WEBAUTHN_DRIVER" env-default:"memory"`

	// RPDisplayName is the Relying Party display name.
	RPDisplayName string `env:"AUTH_WEBAUTHN_RP_DISPLAY_NAME" env-default:"MyApp"`

	// RPID is the Relying Party ID (effective domain).
	RPID string `env:"AUTH_WEBAUTHN_RP_ID" env-default:"localhost"`

	// RPOrigin is the origin for WebAuthn requests.
	RPOrigin string `env:"AUTH_WEBAUTHN_RP_ORIGIN" env-default:"http://localhost:8080"`
}

// User represents a WebAuthn user.
type User interface {
	WebAuthnID() []byte
	WebAuthnName() string
	WebAuthnDisplayName() string
	WebAuthnIcon() string
	WebAuthnCredentials() []Credential
}

// Credential represents a registered WebAuthn credential.
type Credential struct {
	ID              []byte
	PublicKey       []byte
	AttestationType string
	Authenticator   Authenticator
}

// Authenticator represents the authenticator device information.
type Authenticator struct {
	AAGUID       []byte
	SignCount    uint32
	CloneWarning bool
}

// Service defines the interface for WebAuthn operations.
type Service interface {
	// BeginRegistration initiates a new credential registration.
	BeginRegistration(ctx context.Context, user User) (interface{}, error)

	// FinishRegistration completes the registration process.
	FinishRegistration(ctx context.Context, user User, sessionData interface{}, responseData interface{}) (*Credential, error)

	// BeginLogin initiates a login flow.
	BeginLogin(ctx context.Context, user User) (interface{}, error)

	// FinishLogin completes the login flow.
	FinishLogin(ctx context.Context, user User, sessionData interface{}, responseData interface{}) (*Credential, error)
}
