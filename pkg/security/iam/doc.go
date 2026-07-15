/*
Package iam provides common types for Identity and Access Management
(User, Token, Credentials).

Bridge note: application authentication and token verification belong in
pkg/auth (IdentityProvider / Verifier and its adapters). The iam/provider
subpackage is a scaffold for an IdP *server* (issue/validate/revoke) and is
not a substitute for pkg/auth client integrations.
*/
package iam
