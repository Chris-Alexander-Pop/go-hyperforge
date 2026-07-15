/*
Package provider implements a scaffold Identity Provider (IdP) server surface:
authenticate, issue/validate/revoke tokens, and create users.

This is NOT the primary application auth API. Prefer pkg/auth.IdentityProvider
and pkg/auth.Verifier for login and token verification in services. Keep this
package for issuer-side experiments; memory adapter only today (no Dex/Keycloak).
*/
package provider
