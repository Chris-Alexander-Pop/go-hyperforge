// Package memory provides an in-memory WebAuthn test double.
//
// This adapter is intentionally NOT FIDO2-compliant. It tracks challenges and
// credentials so unit tests can exercise Begin/Finish registration and login
// round-trips without a browser or authenticator.
//
// For production, use webauthn/adapters/library (github.com/go-webauthn/webauthn).
// Calling Finish* with a mismatched challenge returns a clear InvalidArgument error.
package memory
