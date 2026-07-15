package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MoneyTestSuite struct {
	test.Suite
}

func (s *MoneyTestSuite) TestNewMoneyNormalizesCurrency() {
	m := commerce.NewMoney(1000, " usd ")
	s.Equal(int64(1000), m.Amount)
	s.Equal("USD", m.Currency)
}

func (s *MoneyTestSuite) TestAddSubSameCurrency() {
	a := commerce.NewMoney(1000, "USD")
	b := commerce.NewMoney(250, "USD")
	sum, err := a.Add(b)
	s.NoError(err)
	s.Equal(int64(1250), sum.Amount)

	diff, err := a.Sub(b)
	s.NoError(err)
	s.Equal(int64(750), diff.Amount)
}

func (s *MoneyTestSuite) TestAddCurrencyMismatch() {
	a := commerce.NewMoney(1000, "USD")
	b := commerce.NewMoney(1000, "EUR")
	_, err := a.Add(b)
	s.Error(err)
}

func (s *MoneyTestSuite) TestFormat() {
	s.Equal("USD 10.00", commerce.Format(commerce.NewMoney(1000, "USD")))
	s.Equal("USD 0.99", commerce.Format(commerce.NewMoney(99, "USD")))
	s.Equal("JPY 1000", commerce.Format(commerce.NewMoney(1000, "JPY")))
	s.Equal("-USD 1.50", commerce.Format(commerce.NewMoney(-150, "USD")))
}

func (s *MoneyTestSuite) TestEqualAndZero() {
	s.True(commerce.NewMoney(0, "USD").IsZero())
	s.True(commerce.NewMoney(100, "usd").Equal(commerce.NewMoney(100, "USD")))
	s.False(commerce.NewMoney(100, "USD").Equal(commerce.NewMoney(100, "EUR")))
}

func TestMoneySuite(t *testing.T) {
	test.Run(t, new(MoneyTestSuite))
}
