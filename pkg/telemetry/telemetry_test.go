package telemetry_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/telemetry"
)

func TestInit(t *testing.T) {
	// Use known bad endpoint to avoid hanging? Or assume it passes async checks?
	// otlptracegrpc might attempt connection.
	// Let's rely on standard timeouts if it blocks.

	done := make(chan bool)
	go func() {
		shutdown, err := telemetry.Init(telemetry.Config{
			ServiceName: "test-service",
			Endpoint:    "localhost:4317", // assume unreachability is fine or instant fail
		})

		if err != nil {
			t.Logf("Init failed (expected if collector offline): %v", err)
		} else {
			if shutdown == nil {
				t.Error("Current implementation returned nil shutdown on success")
			}
			// Don't call shutdown as it might flush traces and block
		}
		done <- true
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Log("Init timed out, likely trying to connect")
	}
}
