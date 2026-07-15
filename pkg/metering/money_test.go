package metering_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	mem "github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateCostMoney(t *testing.T) {
	r := mem.New()
	ctx := context.Background()
	money, err := metering.CalculateCostMoney(ctx, r, metering.UsageEvent{
		TenantID:     "t1",
		ResourceType: "storage.standard",
		Quantity:     3, // GB-months @ 0.10 → 0.30 USD = 30 cents
		Timestamp:    time.Now().UTC(),
	})
	require.NoError(t, err)
	assert.True(t, money.Equal(commerce.NewMoney(30, "USD")))
}
