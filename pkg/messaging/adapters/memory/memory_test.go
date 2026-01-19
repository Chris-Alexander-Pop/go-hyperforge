package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging/tests"
)

func TestMemoryBroker(t *testing.T) {
	broker := memory.New(memory.Config{BufferSize: 100})
	defer broker.Close()

	tests.RunBrokerTests(t, broker)
}
