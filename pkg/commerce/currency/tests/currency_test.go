package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/currency"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/currency/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type CurrencyTestSuite struct {
	test.Suite
	conv currency.Converter
}

func (s *CurrencyTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.conv = memory.New()
}

func (s *CurrencyTestSuite) TestConvert() {
	res, err := s.conv.Convert(s.Ctx, 100.0, "USD", "EUR")
	s.NoError(err)
	s.Equal(85.0, res.ToAmount)
	s.Equal(0.85, res.Rate)

	res2, err := s.conv.Convert(s.Ctx, 85.0, "EUR", "USD")
	s.NoError(err)
	s.InDelta(100.0, res2.ToAmount, 0.001)
}

func (s *CurrencyTestSuite) TestUnknownCurrency() {
	_, err := s.conv.Convert(s.Ctx, 100.0, "USD", "XXX")
	s.Error(err)
	s.Equal(currency.ErrUnsupportedCurrency, err)
}

func (s *CurrencyTestSuite) TestFormatHelpers() {
	s.Equal("USD 10.00", currency.FormatMoney(commerce.NewMoney(1000, "USD")))
	s.Equal("USD 10.00", currency.FormatAmount(10.0, "USD"))
}

func TestCurrencySuite(t *testing.T) {
	test.Run(t, new(CurrencyTestSuite))
}
