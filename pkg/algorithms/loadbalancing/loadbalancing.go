package loadbalancing

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

var (
	// ErrNoNodes is returned when no nodes are available to handle the request.
	ErrNoNodes = errors.FailedPrecondition("no nodes available", nil)
)

// Balancer chooses a node from a list of available nodes.
type Balancer interface {
	// Next returns the next node to use.
	Next(ctx context.Context) (string, error)
	// Add adds a node (with optional weight).
	Add(node string, weight int)
	// Remove removes a node.
	Remove(node string)
}
