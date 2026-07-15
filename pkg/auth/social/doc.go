// Package social provides OAuth2 social login providers (Google, GitHub, Facebook, Apple).
//
// Providers follow the golang.org/x/oauth2 client pattern. Apple uses
// golang.org/x/oauth2/endpoints.Apple and reads identity from the id_token
// returned by Apple's token endpoint (Sign in with Apple does not expose a
// classic userinfo URL). Callers must supply Apple's JWT client secret as
// clientSecret — this package does not mint the Apple client-secret JWT.
package social
