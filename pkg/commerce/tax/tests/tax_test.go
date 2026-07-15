package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
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

func (s *TaxTestSuite) TestCalculateTaxUSNY() {
	res, err := s.calc.CalculateTax(s.Ctx, commerce.NewMoney(10000, "USD"), tax.Location{Country: "US", State: "NY"})
	s.NoError(err)
	s.NotNil(res)
	// NY: state 4% + city 4.5% = 8.5%
	s.InDelta(0.085, res.Rate, 0.0001)
	s.Equal(int64(850), res.TotalTax.Amount)
	s.Equal("US", res.Jurisdiction.Country)
	s.Equal("NY", res.Jurisdiction.State)
	s.Contains(res.Breakdown, "state")
	s.Contains(res.Breakdown, "city")
}

func (s *TaxTestSuite) TestCalculateTaxCountryFallback() {
	res, err := s.calc.CalculateTax(s.Ctx, commerce.NewMoney(10000, "USD"), tax.Location{Country: "US", State: "FL"})
	s.NoError(err)
	// Falls back to US country entry (5% state default)
	s.InDelta(0.05, res.Rate, 0.0001)
	s.Equal(int64(500), res.TotalTax.Amount)
}

func (s *TaxTestSuite) TestUnknownJurisdictionZero() {
	res, err := s.calc.CalculateTax(s.Ctx, commerce.NewMoney(10000, "USD"), tax.Location{Country: "ZZ"})
	s.NoError(err)
	s.Equal(int64(0), res.TotalTax.Amount)
}

func (s *TaxTestSuite) TestInvalidAmount() {
	_, err := s.calc.CalculateTax(s.Ctx, commerce.NewMoney(-1, "USD"), tax.Location{Country: "US"})
	s.Equal(tax.ErrInvalidAmount, err)
}

func TestTaxSuite(t *testing.T) {
	test.Run(t, new(TaxTestSuite))
}
