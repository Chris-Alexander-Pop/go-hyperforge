package wheel_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/timer/wheel"
	"sync"
	"testing"
	"time"
)

func TestHashedWheelTimer(t *testing.T) {
	// Fast tick for testing
	timer := wheel.New(10*time.Millisecond, 10)
	timer.Start()
	defer timer.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	start := time.Now()
	timer.Schedule(50*time.Millisecond, func() {
		wg.Done()
	})

	wg.Wait()
	elapsed := time.Since(start)

	if elapsed < 40*time.Millisecond {
		t.Errorf("wheel.Timer fired too early: %v", elapsed)
	}
	// Allow loose upper bound for CI/local variance
}
