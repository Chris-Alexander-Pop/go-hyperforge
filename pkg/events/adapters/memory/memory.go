// Package memory provides an in-process event bus backed by Go memory.
//
// It is suitable for unit tests and single-process domain event fan-out.
// Handlers run synchronously by default (Publish returns aggregated errors).
// Set Config.Async to dispatch via a bounded concurrency.WorkerPool.
//
// Topics should be domain names (e.g. "users", "orders"), not event types.
//
// Usage:
//
//	bus := memory.New(events.Config{})
//	defer bus.Close()
//
//	sub, err := bus.Subscribe(ctx, "users", handler)
//	if err != nil {
//	    return err
//	}
//	defer bus.Unsubscribe(ctx, sub)
//
//	err = bus.Publish(ctx, "users", events.Event{Type: "user.created", Payload: user})
package memory

import (
	"context"
	stderrors "errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/google/uuid"
)

// Ensure MemoryBus implements events.Bus at compile time.
var _ events.Bus = (*MemoryBus)(nil)

type subscription struct {
	id      events.Subscription
	topic   string
	handler events.Handler
}

// MemoryBus is an in-memory implementation of events.Bus.
type MemoryBus struct {
	config events.Config
	mu     *concurrency.SmartRWMutex

	subsByTopic map[string][]subscription
	subsByID    map[events.Subscription]subscription

	closed atomic.Bool
	// inflight tracks Publish calls still enqueueing or running sync handlers.
	inflight sync.WaitGroup
	// wg tracks async handler tasks still executing.
	wg sync.WaitGroup

	pool       *concurrency.WorkerPool
	poolCancel context.CancelFunc
}

// New creates a MemoryBus with the given configuration.
// Zero-valued Config fields receive package defaults.
func New(cfg events.Config) *MemoryBus {
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 256
	}

	m := &MemoryBus{
		config:      cfg,
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "MemoryBus"}),
		subsByTopic: make(map[string][]subscription),
		subsByID:    make(map[events.Subscription]subscription),
	}

	if cfg.Async {
		poolCtx, cancel := context.WithCancel(context.Background())
		m.poolCancel = cancel
		m.pool = concurrency.NewWorkerPool(cfg.Workers, cfg.QueueSize)
		m.pool.Start(poolCtx)
	}

	return m
}

// Publish delivers an event to all subscribers of topic.
// In sync mode (default), handlers run with ctx and errors are aggregated.
// In async mode, handlers are submitted to a bounded worker pool; Publish returns
// after enqueue (handler errors are logged).
func (m *MemoryBus) Publish(ctx context.Context, topic string, event events.Event) error {
	if m.closed.Load() {
		return events.ErrClosed(nil)
	}
	if topic == "" {
		return events.ErrInvalidTopic(topic, nil)
	}
	if event.Type == "" {
		return events.ErrInvalidEvent("event type is required", nil)
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	m.inflight.Add(1)
	defer m.inflight.Done()

	m.mu.RLock()
	if m.closed.Load() {
		m.mu.RUnlock()
		return events.ErrClosed(nil)
	}
	subs := append([]subscription(nil), m.subsByTopic[topic]...)
	m.mu.RUnlock()

	if len(subs) == 0 {
		return nil
	}

	if m.config.Async {
		return m.publishAsync(ctx, topic, event, subs)
	}
	return m.publishSync(ctx, event, subs)
}

func (m *MemoryBus) publishSync(ctx context.Context, event events.Event, subs []subscription) error {
	var errs []error
	for _, sub := range subs {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		if err := sub.handler(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return events.ErrHandlerFailed(stderrors.Join(errs...))
}

func (m *MemoryBus) publishAsync(ctx context.Context, topic string, event events.Event, subs []subscription) error {
	for _, sub := range subs {
		if m.closed.Load() {
			return events.ErrClosed(nil)
		}
		sub := sub
		m.wg.Add(1)
		m.pool.Submit(func(_ context.Context) {
			defer m.wg.Done()
			if err := sub.handler(ctx, event); err != nil {
				logger.L().ErrorContext(ctx, "event handler failed",
					"topic", topic,
					"type", event.Type,
					"id", event.ID,
					"subscription", string(sub.id),
					"error", err,
				)
			}
		})
	}
	return nil
}

// Subscribe registers a handler for topic and returns a subscription ID.
func (m *MemoryBus) Subscribe(ctx context.Context, topic string, handler events.Handler) (events.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if m.closed.Load() {
		return "", events.ErrClosed(nil)
	}
	if topic == "" {
		return "", events.ErrInvalidTopic(topic, nil)
	}
	if handler == nil {
		return "", events.ErrInvalidEvent("handler is required", nil)
	}

	id := events.Subscription(uuid.NewString())
	sub := subscription{id: id, topic: topic, handler: handler}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed.Load() {
		return "", events.ErrClosed(nil)
	}

	m.subsByTopic[topic] = append(m.subsByTopic[topic], sub)
	m.subsByID[id] = sub
	return id, nil
}

// Unsubscribe removes the subscription identified by id.
func (m *MemoryBus) Unsubscribe(ctx context.Context, id events.Subscription) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.closed.Load() {
		return events.ErrClosed(nil)
	}
	if id == "" {
		return events.ErrSubscriptionNotFound(string(id), nil)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed.Load() {
		return events.ErrClosed(nil)
	}

	sub, ok := m.subsByID[id]
	if !ok {
		return events.ErrSubscriptionNotFound(string(id), nil)
	}

	delete(m.subsByID, id)

	topicSubs := m.subsByTopic[sub.topic]
	for i, s := range topicSubs {
		if s.id == id {
			m.subsByTopic[sub.topic] = append(topicSubs[:i], topicSubs[i+1:]...)
			break
		}
	}
	if len(m.subsByTopic[sub.topic]) == 0 {
		delete(m.subsByTopic, sub.topic)
	}
	return nil
}

// Close marks the bus closed, waits for in-flight work, and stops the worker pool.
func (m *MemoryBus) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}

	m.mu.Lock()
	m.subsByTopic = make(map[string][]subscription)
	m.subsByID = make(map[events.Subscription]subscription)
	m.mu.Unlock()

	// Wait for concurrent Publish calls to finish enqueueing/running.
	m.inflight.Wait()
	// Wait for async handlers already submitted to the pool.
	m.wg.Wait()

	if m.pool != nil {
		m.pool.Stop()
	}
	if m.poolCancel != nil {
		m.poolCancel()
	}
	return nil
}
