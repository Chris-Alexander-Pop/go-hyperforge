package messaging_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/testsuite"
)

// TestMemoryBroker runs the test suite against the in-memory broker.
func TestMemoryBroker(t *testing.T) {
	broker := memory.New(memory.Config{BufferSize: 100})
	defer broker.Close()

	testsuite.RunBrokerTests(t, broker)
}
