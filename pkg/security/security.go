package security

// Provider identifiers shared across security subpackages.
// Concrete adapters live under each subpackage's adapters/ tree.
// Today most backends are memory-only; cloud/Vault drivers are reserved names.
const (
	ProviderMemory = "memory"

	// Captcha providers
	ProviderRecaptcha = "recaptcha"
	ProviderHCaptcha  = "hcaptcha"
	ProviderTurnstile = "turnstile"

	// Secrets providers (reserved unless an adapter exists)
	ProviderVault             = "vault"
	ProviderAWSSecretsManager = "aws-secrets-manager"
	ProviderGCPSecretManager  = "gcp-secret-manager"
	ProviderAzureKeyVault     = "azure-key-vault"

	// KMS providers (reserved unless an adapter exists)
	ProviderAWSKMS   = "aws-kms"
	ProviderGCPKMS   = "gcp-kms"
	ProviderAzureKMS = "azure-kms"

	// Fraud / WAF / scanning (reserved)
	ProviderMaxMind    = "maxmind"
	ProviderAWSWAF     = "aws-waf"
	ProviderCloudflare = "cloudflare"
	ProviderClamAV     = "clamav"
	ProviderGuardDuty  = "guardduty"
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
