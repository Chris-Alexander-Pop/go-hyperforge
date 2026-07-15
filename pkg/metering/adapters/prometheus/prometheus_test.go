package prometheus_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	mem "github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusExporter_RecordAndGather(t *testing.T) {
	exp := prometheus.New()
	ctx := context.Background()

	require.NoError(t, exp.RecordUsage(ctx, metering.UsageEvent{
		TenantID:     "t1",
		ResourceType: "compute.instance.small",
		ResourceID:   "i-1",
		Quantity:     2.5,
		Timestamp:    time.Now().UTC(),
	}))
	require.NoError(t, exp.RecordUsage(ctx, metering.UsageEvent{
		TenantID:     "t1",
		ResourceType: "compute.instance.small",
		ResourceID:   "i-2",
		Quantity:     1.5,
		Timestamp:    time.Now().UTC(),
	}))

	var buf bytes.Buffer
	require.NoError(t, exp.Gather(&buf))
	text := buf.String()
	assert.Contains(t, text, "metering_usage_quantity_total")
	assert.Contains(t, text, `tenant_id="t1"`)
	assert.Contains(t, text, "4") // 2.5+1.5
	assert.Contains(t, text, "metering_usage_events_total 2")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	exp.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, strings.Contains(rr.Body.String(), "metering_usage_quantity_total"))

	events, err := exp.GetUsage(ctx, metering.UsageFilter{TenantID: "t1"})
	require.NoError(t, err)
	require.Len(t, events, 2)
}

func TestCalculateCostMoney_WiresBilling(t *testing.T) {
	r := mem.New()
	ctx := context.Background()
	money, err := metering.CalculateCostMoney(ctx, r, metering.UsageEvent{
		TenantID:     "t1",
		ResourceType: "compute.instance.small",
		Quantity:     10, // hours
		Timestamp:    time.Now().UTC(),
	})
	require.NoError(t, err)
	// default rate 0.02 USD/hour * 10 = 0.20 USD = 20 cents
	assert.True(t, money.Equal(commerce.NewMoney(20, "USD")))
}
