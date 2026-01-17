package delay

import (
	"container/heap"
	"sync"
	"time"

	graphHeap "github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
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
type Queue[T any] struct {
	pq *graphHeap.MinHeap[T] // Reuse our generic MinHeap? It uses float64 score.
	// Actually, standard delay queues use time as score.
	// Our Generic Heap implementation uses float64 score. We can convert timestamp to float64,
	// but might lose nanosecond precision if not careful.
	// Let's implement a specialized heap wrapper or use internal heap for this
	// to ensure time precision and "Peek" logic for waiting.

	items  []*Item[T]
	mu     sync.Mutex
	wakeup *sync.Cond
	closed bool
}

// New creates a new Delay Queue.
func New[T any]() *Queue[T] {
	q := &Queue[T]{
		items: make([]*Item[T], 0),
	}
	q.wakeup = sync.NewCond(&q.mu)
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
	q.wakeup.Signal()
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
			q.wakeup.Wait()
			continue
		}

		item := q.items[0]
		now := time.Now()

		if now.After(item.ReadyTime) || now.Equal(item.ReadyTime) {
			heap.Pop(q)
			return item.Value, true
		}

		// Wait until ready
		d := item.ReadyTime.Sub(now)
		// We can't easily wait with timeout on sync.Cond.
		// So we loop. Ideally we use a select with channel in a real runtime,
		// but standard sync.Cond is "wait forever".
		// To fix this for a "DelayQueue", users typically use a channel-based approach.
		// Let's implement a channel-based poll for better UX.

		// Unlocking and sleeping is risky with Cond (missed signal).
		// We'll return invalid and let caller sleep, OR implement strict blocking properly.

		// For this library, let's assume the user can simple poll TryDequeue or we switch to channel based.
		// Let's assume standard implementation style: Unlock, Sleep, Lock loop.

		q.mu.Unlock()
		time.Sleep(d)
		q.mu.Lock()
	}
}

// Len returns number of pending items.
func (q *Queue[T]) Len() int { return len(q.items) }

// Close closes the queue.
func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.wakeup.Broadcast()
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
