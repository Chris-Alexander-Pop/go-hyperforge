package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/tax"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/tax/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type TaxTestSuite struct {
	test.Suite
	calc tax.Calculator
}

func (s *TaxTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.calc = memory.New()
}

func (s *TaxTestSuite) TestCalculateTax() {
	res, err := s.calc.CalculateTax(s.Ctx, 100.0, tax.Location{Country: "US", State: "NY"})
	s.NoError(err)
	s.NotNil(res)
	s.Equal(10.0, res.TotalTax)
	s.Equal(0.10, res.Rate)

	res2, err := s.calc.CalculateTax(s.Ctx, 100.0, tax.Location{Country: "CA"})
	s.NoError(err)
	s.Equal(0.0, res2.TotalTax)
}

func TestTaxSuite(t *testing.T) {
	test.Run(t, new(TaxTestSuite))
}
