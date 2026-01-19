package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/telemetry"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type TelemetryTestSuite struct {
	test.Suite
}

func (s *TelemetryTestSuite) TestInit() {
	cfg := telemetry.Config{
		ServiceName: "test-service",
		Endpoint:    "localhost:4317", // No listener needed for setup
	}

	shutdown, err := telemetry.Init(cfg)
	s.NoError(err)
	s.NotNil(shutdown)

	// Verify shutdown works (doesn't hang/crash)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = shutdown(ctx)
	// It might error due to connection refused, but shouldn't panic
	// We check that it returns (error is acceptable in unit test environment)
	_ = err
}

func TestTelemetrySuite(t *testing.T) {
	test.Run(t, new(TelemetryTestSuite))
}
