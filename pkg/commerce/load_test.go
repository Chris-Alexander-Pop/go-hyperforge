package commerce_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type LoadConfigSuite struct {
	test.Suite
}

func (s *LoadConfigSuite) TestLoadConfigDefaults() {
	cfg, err := commerce.LoadConfig()
	s.NoError(err)
	s.Equal("memory", cfg.PaymentProvider)
	s.Equal("memory", cfg.BillingProvider)
	s.Equal("memory", cfg.TaxProvider)
	s.Equal("memory", cfg.CurrencyProvider)
}

func TestLoadConfigSuite(t *testing.T) {
	test.Run(t, new(LoadConfigSuite))
}
