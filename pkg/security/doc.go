/*
Package security provides security services and integrations.

Subpackages (adapters noted):

  - captcha: CAPTCHA verification (memory + reCAPTCHA HTTP)
  - crypto: AES-GCM, hashing, envelope encryption; PQC is experimental
  - crypto/kms: Key management (memory + AWS KMS Encrypt/Decrypt)
  - fraud: Fraud detection / risk scoring (memory)
  - iam: Shared IAM types; provider is a scaffold IdP — prefer pkg/auth for app auth
  - scanning: Malware / vulnerability scanning (memory)
  - secrets: Secret management (memory + HashiCorp Vault KV v2 HTTP)
  - waf: Web Application Firewall control (memory + Cloudflare IP access rules)

Honesty note: GCP/Azure KMS, AWS WAF, GuardDuty, Dilithium signatures, and cloud
secret managers beyond Vault are not wired. Those names remain reserved in
Provider constants. Prefer memory adapters for unit tests; use Vault / AWS KMS /
Cloudflare adapters when targeting those control planes.

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
