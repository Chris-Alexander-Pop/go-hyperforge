// Package sticky implements session-affinity (sticky) load balancing.
//
// NextKey maps a session key to a backend and remembers the assignment until
// the backend is removed. New sessions are placed via optional fallback or
// round-robin over the current node set.
package sticky

import (
	"context"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
)

// contextKey is the type for session keys stored in context.
type contextKey struct{}

// SessionKey is the context key for an optional session identifier used by Next.
var SessionKey = contextKey{}

// WithSession returns a child context carrying the session key for Next().
func WithSession(ctx context.Context, session string) context.Context {
	return context.WithValue(ctx, SessionKey, session)
}

// SessionFromContext extracts a session key previously stored with WithSession.
func SessionFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(SessionKey).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// Balancer implements session-affinity load balancing.
type Balancer struct {
	mu       sync.RWMutex
	nodes    []string
	affinity map[string]string // session -> node
	rr       int               // round-robin cursor for new sessions
	fallback loadbalancing.Balancer
}

// New creates a sticky balancer. fallback may be nil (round-robin placement).
func New(fallback loadbalancing.Balancer, nodes ...string) *Balancer {
	b := &Balancer{
		nodes:    append([]string(nil), nodes...),
		affinity: make(map[string]string),
		fallback: fallback,
	}
	return b
}

var _ loadbalancing.Balancer = (*Balancer)(nil)

// Next returns the backend for the session key in ctx (via WithSession), or
// places a new anonymous request via fallback/round-robin when no session is set.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	if session, ok := SessionFromContext(ctx); ok {
		return b.NextKey(ctx, session)
	}
	return b.placeNew(ctx)
}

// NextKey returns the sticky backend for session. The same session always maps
// to the same node until that node is removed.
func (b *Balancer) NextKey(ctx context.Context, session string) (string, error) {
	_ = ctx
	if session == "" {
		return b.placeNew(ctx)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if node, ok := b.affinity[session]; ok {
		if b.hasNodeLocked(node) {
			return node, nil
		}
		delete(b.affinity, session)
	}

	node, err := b.placeNewLocked(ctx)
	if err != nil {
		return "", err
	}
	b.affinity[session] = node
	return node, nil
}

func (b *Balancer) placeNew(ctx context.Context) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.placeNewLocked(ctx)
}

func (b *Balancer) placeNewLocked(ctx context.Context) (string, error) {
	if len(b.nodes) == 0 {
		return "", loadbalancing.ErrNoNodes
	}
	// Prefer round-robin over the sticky node set. Optional fallback is only
	// consulted when set and the sticky set is empty of usable nodes (should
	// not happen when len(nodes)>0); kept for composition with Maglev/P2C.
	if b.fallback != nil && len(b.nodes) == 0 {
		fb := b.fallback
		b.mu.Unlock()
		node, err := fb.Next(ctx)
		b.mu.Lock()
		return node, err
	}
	_ = ctx
	node := b.nodes[b.rr%len(b.nodes)]
	b.rr++
	return node, nil
}

func (b *Balancer) hasNodeLocked(node string) bool {
	for _, n := range b.nodes {
		if n == node {
			return true
		}
	}
	return false
}

// Add adds a backend (weight ignored; affinity is 1:1).
func (b *Balancer) Add(node string, weight int) {
	_ = weight
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.hasNodeLocked(node) {
		return
	}
	b.nodes = append(b.nodes, node)
	if b.fallback != nil {
		b.fallback.Add(node, weight)
	}
}

// Remove removes a backend and clears affinity entries pointing at it.
func (b *Balancer) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, n := range b.nodes {
		if n == node {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			break
		}
	}
	for session, n := range b.affinity {
		if n == node {
			delete(b.affinity, session)
		}
	}
	if b.fallback != nil {
		b.fallback.Remove(node)
	}
}

// AffinitySize returns the number of remembered session→node mappings (tests).
func (b *Balancer) AffinitySize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.affinity)
}
