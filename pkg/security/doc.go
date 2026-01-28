/*
Package security provides security services and integrations.

Subpackages:

  - captcha: CAPTCHA verification (reCAPTCHA, hCaptcha)
  - crypto/kms: Key Management Services (AWS KMS, GCP KMS, Azure Key Vault)
  - fraud: Fraud detection services
  - scanning: Security scanning (malware, vulnerabilities)
  - secrets: Secret management (this is the canonical location)
  - waf: Web Application Firewall

Note: This package's secrets subpackage is the canonical location for
secret management. The pkg/secrets package delegates to this one.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/security/secrets"

	vault, err := hashicorp.New(cfg)
	secret, err := vault.Get(ctx, "database/password")
*/
package security
