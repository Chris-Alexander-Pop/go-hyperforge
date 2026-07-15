package commerce

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
)

// Config is the root env-tagged configuration for commerce capability providers.
// Subpackages retain their own Config types; this aggregates common provider selection.
type Config struct {
	// PaymentProvider selects payment backend: memory, stripe, paypal.
	PaymentProvider string `env:"PAYMENT_PROVIDER" env-default:"memory"`

	// BillingProvider selects billing backend: memory, stripe.
	BillingProvider string `env:"BILLING_PROVIDER" env-default:"memory"`

	// TaxProvider selects tax backend: memory, taxjar, avalara.
	TaxProvider string `env:"TAX_PROVIDER" env-default:"memory"`

	// CurrencyProvider selects FX backend: memory, openexchangerates, frankfurter.
	CurrencyProvider string `env:"CURRENCY_PROVIDER" env-default:"memory"`
}

// LoadConfig loads commerce.Config via pkg/config (env / optional .env) and validates it.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
