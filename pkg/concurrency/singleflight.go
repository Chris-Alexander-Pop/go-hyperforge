package concurrency

import "sync"

// call is an in-flight or completed Do call.
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group coalesces concurrent work for the same key into a single execution
// (golang.org/x/sync/singleflight style).
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do executes and returns the results of the given function, making sure that
// only one execution is in-flight for a given key at a time. If a duplicate
// comes in while the original is still executing, the duplicate waits for the
// original and receives the same results.
//
// The shared return value reports whether v was given to multiple callers.
func (g *Group) Do(key string, fn func() (interface{}, error)) (v interface{}, err error, shared bool) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err, true
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err, false
}

// Forget tells the Group to forget about a key so a subsequent Do for that key
// will call fn rather than waiting for an in-flight call.
func (g *Group) Forget(key string) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}
