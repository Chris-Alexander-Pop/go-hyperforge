// Package jwt provides a local JWT (JSON Web Token) authentication adapter.
//
// This package implements the auth.Verifier interface and provides functionality
// for generating and verifying HMAC-SHA256 signed tokens. It is designed for
// services that need to issue and validate their own tokens without relying on
// an external Identity Provider (IdP).
//
// # Configuration
//
// The package is configured via the Config struct, which supports environment
// variable loading:
//
//   - Secret: The shared secret key for signing tokens (Required, env: JWT_SECRET)
//   - Expiration: Duration until token expiry (Default: 24h, env: JWT_EXPIRATION)
//   - Issuer: The issuer claim value (Default: system-design-library, env: JWT_ISSUER)
//
// # Usage
//
// To use the adapter, initialize it with a configuration and use the Generate
// method to create tokens and the Verify method to validate them.
//
// Example:
//
//	cfg := jwt.Config{
//		Secret:     "your-256-bit-secret",
//		Expiration: time.Hour,
//		Issuer:     "my-app",
//	}
//	adapter := jwt.New(cfg)
//
//	// Generate a token
//	token, err := adapter.Generate("user-123", "admin")
//	if err != nil {
//		// handle error
//	}
//
//	// Verify a token
//	claims, err := adapter.Verify(context.Background(), token)
//	if err != nil {
//		// handle error
//	}
//	fmt.Printf("User: %s, Role: %s\n", claims.Subject, claims.Role)
//
// # Security
//
// This adapter uses HMAC-SHA256 (HS256) for signing. Ensure that the Secret
// is sufficiently strong and kept confidential.
package jwt
