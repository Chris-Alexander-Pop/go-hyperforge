package security

// Provider identifiers shared across security subpackages.
// Concrete adapters live under each subpackage's adapters/ tree.
// Implemented today: memory, vault (secrets), aws-kms, cloudflare (WAF), recaptcha.
// Other cloud names remain reserved placeholders.
const (
	ProviderMemory = "memory"

	// Captcha providers
	ProviderRecaptcha = "recaptcha" // adapters/recaptcha
	ProviderHCaptcha  = "hcaptcha"  // reserved
	ProviderTurnstile = "turnstile" // reserved

	// Secrets providers
	ProviderVault             = "vault"               // adapters/vault (KV v2)
	ProviderAWSSecretsManager = "aws-secrets-manager" // reserved
	ProviderGCPSecretManager  = "gcp-secret-manager"  // reserved
	ProviderAzureKeyVault     = "azure-key-vault"     // reserved

	// KMS providers
	ProviderAWSKMS   = "aws-kms"   // adapters/awskms
	ProviderGCPKMS   = "gcp-kms"   // reserved
	ProviderAzureKMS = "azure-kms" // reserved

	// Fraud / WAF / scanning
	ProviderMaxMind    = "maxmind"    // reserved
	ProviderAWSWAF     = "aws-waf"    // reserved
	ProviderCloudflare = "cloudflare" // waf/adapters/cloudflare
	ProviderClamAV     = "clamav"     // reserved
	ProviderGuardDuty  = "guardduty"  // reserved
)
// Domain names for documentation and driver registries.
const (
	DomainCaptcha  = "captcha"
	DomainCrypto   = "crypto"
	DomainFraud    = "fraud"
	DomainIAM      = "iam"
	DomainKMS      = "kms"
	DomainScanning = "scanning"
	DomainSecrets  = "secrets"
	DomainWAF      = "waf"
)
