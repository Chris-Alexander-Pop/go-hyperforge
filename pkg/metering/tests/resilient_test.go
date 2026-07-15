package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	metermem "github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/memory"
)

func TestResilientMeter_Record(t *testing.T) {
	inner := metermem.New()
	t.Cleanup(func() { _ = inner.Close() })
	meter := metering.NewResilientMeter(inner, metering.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	err := meter.RecordUsage(t.Context(), metering.UsageEvent{
		TenantID:     "t1",
		ResourceType: "compute.instance.small",
		ResourceID:   "i-1",
		Quantity:     1,
		Timestamp:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
}
