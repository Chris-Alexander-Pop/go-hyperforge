package events_test

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/chris-alexander-pop/system-design-library/pkg/events/adapters/memory"
)

func Example() {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()

	handler := func(ctx context.Context, event events.Event) error {
		fmt.Printf("Received: %s\n", event.Type)
		return nil
	}

	sub, _ := bus.Subscribe(ctx, "users", handler)
	defer bus.Unsubscribe(ctx, sub)

	event := events.Event{
		ID:        "evt-123",
		Type:      "user.created",
		Source:    "user-service",
		Timestamp: time.Now(),
		Payload:   map[string]string{"user_id": "123", "email": "alice@example.com"},
	}

	_ = bus.Publish(ctx, "users", event)
	// Output: Received: user.created
}

func ExampleEvent() {
	event := events.Event{
		ID:        "evt-456",
		Type:      "order.placed",
		Source:    "order-service",
		Timestamp: time.Now().UTC(),
		Payload: map[string]interface{}{
			"order_id": "ord-789",
			"total":    99.99,
			"items":    3,
		},
	}

	fmt.Println(event.Type)
	// Output: order.placed
}
