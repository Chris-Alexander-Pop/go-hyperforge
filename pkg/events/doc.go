/*
Package events provides an in-process event bus for decoupling components via domain events.

It defines a standard Event structure and a Bus interface for Publish/Subscribe patterns.
This package is intended for local process constraints. For distributed messaging, see pkg/messaging.

Usage:

	bus := memory.New()
	bus.Subscribe(ctx, "user.created", func(ctx context.Context, e events.Event) error {
	    // Handle event
	    return nil
	})

	bus.Publish(ctx, "user.created", events.Event{Type: "user.created", Payload: user})
*/
package events
