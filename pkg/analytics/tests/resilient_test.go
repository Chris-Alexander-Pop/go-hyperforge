package analytics_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	analyticsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/memory"
)

func TestResilientTracker_AddCount(t *testing.T) {
	inner, err := analyticsmem.New(analytics.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = inner.Close() })
	tracker := analytics.NewResilientTracker(inner, analytics.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	if err := tracker.Add(t.Context(), "users", "u1"); err != nil {
		t.Fatal(err)
	}
	n, err := tracker.Count(t.Context(), "users")
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("expected count > 0")
	}
}
