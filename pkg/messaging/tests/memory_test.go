package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging/adapters/memory"
)

// TestMemoryBroker runs the test suite against the in-memory broker.
func TestMemoryBroker(t *testing.T) {
	broker := memory.New(memory.Config{BufferSize: 100})
	defer broker.Close()

	RunBrokerTests(t, broker)
}
