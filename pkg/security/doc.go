/*
Package security provides security services and integrations.

Subpackages (adapters noted):

  - captcha: CAPTCHA verification (memory + reCAPTCHA HTTP)
  - crypto: AES-GCM, hashing, envelope encryption; PQC ML-KEM + ML-DSA via CIRCL
  - crypto/kms: Key management (memory + AWS/GCP/Azure KMS Encrypt/Decrypt)
  - fraud: Fraud detection / risk scoring (memory)
  - iam: Shared IAM types; provider is a scaffold IdP — prefer pkg/auth for app auth
  - scanning: Malware / vulnerability scanning (memory + GuardDuty + ClamAV)
  - secrets: Secret management (memory + Vault + AWS/GCP/Azure Key Vault)
  - waf: Web Application Firewall control (memory + Cloudflare + AWS WAFv2)

Honesty note: Prefer memory adapters for unit tests; use Vault / cloud KMS /
secret managers / WAF / ClamAV / GuardDuty adapters when targeting those
control planes. Some Provider constant names remain reserved for future backends.

Bridge with pkg/auth:

  - pkg/auth.IdentityProvider / Verifier — client-side login and token verify
    for application services (JWT, OIDC, sessions, cloud IdPs).
  - pkg/security/iam/provider.IdentityProvider — scaffold for an IdP *server*
    (issue/validate/revoke). Do not treat it as a drop-in for pkg/auth;
    compose auth clients against pkg/auth, and use iam/provider only when
    building an issuer-side component.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets/adapters/memory"

	mgr := memory.New()
	_ = mgr.Set(ctx, "database/password", "s3cr3t")
*/
package security
