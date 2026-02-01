package delay

import (
	"container/heap"
	"sync"
	"time"
)

// Item represents a delayed task.
type Item[T any] struct {
	Value     T
	Priority  int64 // timestamp in unix nanos or simple priority
	Index     int
	ReadyTime time.Time
}

// Queue implements a thread-safe delay queue.
// Items are dequeued only after their ReadyTime has passed.
// Uses container/heap internally for time precision (avoiding float64 score conversion).
type Queue[T any] struct {
	items    []*Item[T]
	mu       sync.Mutex
	notifyCh chan struct{}
	closed   bool
}

// New creates a new Delay Queue.
func New[T any]() *Queue[T] {
	q := &Queue[T]{
		items:    make([]*Item[T], 0),
		notifyCh: make(chan struct{}, 1),
	}
	return q
}

// Enqueue adds an item with a delay.
func (q *Queue[T]) Enqueue(value T, delay time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	readyTime := time.Now().Add(delay)
	item := &Item[T]{
		Value:     value,
		ReadyTime: readyTime,
		Priority:  readyTime.UnixNano(),
	}
	heap.Push(q, item)

	// Signal new item
	if !q.closed {
		select {
		case q.notifyCh <- struct{}{}:
		default:
		}
	}
}

// Dequeue blocks until an item is ready.
func (q *Queue[T]) Dequeue() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		if q.closed {
			var zero T
			return zero, false
		}

		if len(q.items) == 0 {
			q.mu.Unlock()
			// Wait for signal (item enqueued or closed)
			<-q.notifyCh
			q.mu.Lock()
			continue
		}

		item := q.items[0]
		now := time.Now()

		if now.After(item.ReadyTime) || now.Equal(item.ReadyTime) {
			heap.Pop(q)

			// Signal if more items remain (baton passing)
			if len(q.items) > 0 {
				select {
				case q.notifyCh <- struct{}{}:
				default:
				}
			}

			return item.Value, true
		}

		// Wait until ready
		d := item.ReadyTime.Sub(now)

		q.mu.Unlock()

		timer := time.NewTimer(d)
		select {
		case <-timer.C:
			// Timer expired, loop back to check
		case <-q.notifyCh:
			// Woken up by new item or close
			if !timer.Stop() {
				// Drain channel if needed
				select {
				case <-timer.C:
				default:
				}
			}
		}

		q.mu.Lock()
	}
}

// Len returns number of pending items.
func (q *Queue[T]) Len() int { return len(q.items) }

// Close closes the queue.
func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.notifyCh)
	}
}

// internal heap interface implementation
func (q *Queue[T]) Less(i, j int) bool {
	return q.items[i].ReadyTime.Before(q.items[j].ReadyTime)
}

func (q *Queue[T]) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
	q.items[i].Index = i
	q.items[j].Index = j
}

func (q *Queue[T]) Push(x interface{}) {
	n := len(q.items)
	item := x.(*Item[T])
	item.Index = n
	q.items = append(q.items, item)
}

func (q *Queue[T]) Pop() interface{} {
	old := q.items
	n := len(old)
	item := old[n-1]
	item.Index = -1
	q.items = old[0 : n-1]
	return item
}
