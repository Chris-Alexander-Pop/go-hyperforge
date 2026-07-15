/*
Package security provides security services and integrations.

Subpackages (memory adapters today unless noted):

  - captcha: CAPTCHA verification (memory + reCAPTCHA HTTP skeleton)
  - crypto: AES-GCM, hashing, envelope encryption; PQC is experimental
  - crypto/kms: Key management interface (memory only; cloud KMS reserved)
  - fraud: Fraud detection / risk scoring (memory)
  - iam: Shared IAM types; provider is a scaffold IdP — prefer pkg/auth for app auth
  - scanning: Malware / vulnerability scanning (memory)
  - secrets: Secret management (memory; Vault/cloud SM reserved)
  - waf: Web Application Firewall control (memory)

Honesty note: production drivers (HashiCorp Vault, AWS/GCP/Azure KMS,
cloud WAF, GuardDuty, Dilithium signatures) are not wired. Names appear in
Config/Provider constants as reserved placeholders. Prefer memory adapters
for tests and bring your own remote adapter behind the interfaces.

Bridge with pkg/auth:

  - pkg/auth.IdentityProvider / Verifier — client-side login and token verify
    for application services (JWT, OIDC, sessions, cloud IdPs).
  - pkg/security/iam/provider.IdentityProvider — scaffold for an IdP *server*
    (issue/validate/revoke). Do not treat it as a drop-in for pkg/auth;
    compose auth clients against pkg/auth, and use iam/provider only when
    building an issuer-side component.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/security/secrets/adapters/memory"

	mgr := memory.New()
	_ = mgr.Set(ctx, "database/password", "s3cr3t")
*/
package security
