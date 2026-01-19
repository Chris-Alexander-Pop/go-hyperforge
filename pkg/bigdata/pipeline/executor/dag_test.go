package executor

import (
	"context"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/events"
)

type MockBus struct {
	published []events.Event
	mu        sync.Mutex
}

func (m *MockBus) Publish(ctx context.Context, topic string, event events.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, event)
	return nil
}

func (m *MockBus) Subscribe(ctx context.Context, topic string, handler events.Handler) error {
	return nil
}

func (m *MockBus) Close() error {
	return nil
}

func TestDAGExecutorEvents(t *testing.T) {
	bus := &MockBus{}
	dag := New(bus)

	// Define DAG: A -> B
	dag.AddTask("A", func(ctx context.Context) error {
		return nil
	})
	dag.AddTask("B", func(ctx context.Context) error {
		return nil
	}, "A")

	ctx := context.Background()
	if err := dag.Run(ctx); err != nil {
		t.Fatalf("DAG execution failed: %v", err)
	}

	// Verify events
	// Should have: A.started, A.completed, B.started, B.completed (Order of A/B distinct, but start/comp pairs)
	// Actually A must complete before B starts.
	// Order: A-Start, A-Comp, B-Start, B-Comp.

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if len(bus.published) != 4 {
		t.Errorf("Expected 4 events, got %d", len(bus.published))
	}

	types := []string{}
	for _, e := range bus.published {
		types = append(types, e.Type)
	}

	// Verify we have started/completed for both tasks
	counts := make(map[string]int)
	for _, typ := range types {
		counts[typ]++
	}

	if counts["bigdata.task.started"] != 2 {
		t.Errorf("Expected 2 started events, got %d", counts["bigdata.task.started"])
	}
	if counts["bigdata.task.completed"] != 2 {
		t.Errorf("Expected 2 completed events, got %d", counts["bigdata.task.completed"])
	}
}
