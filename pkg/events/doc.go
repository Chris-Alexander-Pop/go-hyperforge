/*
Package events provides an in-process event bus for decoupling components via domain events.

It defines a standard Event structure and a Bus interface for Publish/Subscribe patterns.
This package is intended for local process constraints. For distributed messaging, see pkg/messaging.

Outbox / NewOutboxBus bridge domain events to pkg/messaging.Producer for durable fan-out
(PACKAGE_STANDARDS §9.5). They are not a transactional outbox store.

Topics use domain-based names (e.g. "users", "orders"). Event types use dot-notation
(e.g. "user.created", "order.placed") — see PACKAGE_STANDARDS §9.

Usage:

	bus := memory.New(events.Config{})
	sub, err := bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error {
	    // Handle event
	    return nil
	})
	if err != nil {
	    return err
	}
	defer bus.Unsubscribe(ctx, sub)

	err = bus.Publish(ctx, "users", events.Event{
	    Type:    "user.created",
	    Source:  "user-service",
	    Payload: user,
	})
*/
package events
