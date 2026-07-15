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
	ProviderAWSSecretsManager = "aws-secrets-manager" // adapters/awssecrets
	ProviderGCPSecretManager  = "gcp-secret-manager"  // adapters/gcpsecretmanager
	ProviderAzureKeyVault     = "azure-key-vault"     // adapters/azurekv

	// KMS providers
	ProviderAWSKMS   = "aws-kms"   // adapters/awskms
	ProviderGCPKMS   = "gcp-kms"   // adapters/gcpkms
	ProviderAzureKMS = "azure-kms" // adapters/azurekms

	// Fraud / WAF / scanning
	ProviderMaxMind    = "maxmind"    // reserved
	ProviderAWSWAF     = "aws-waf"    // waf/adapters/aws
	ProviderCloudflare = "cloudflare" // waf/adapters/cloudflare
	ProviderClamAV     = "clamav"     // scanning/adapters/clamav
	ProviderGuardDuty  = "guardduty"  // scanning/adapters/guardduty
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
