// Package delay provides a thread-safe delay queue implementation.
//
// The Queue[T] type allows enqueueing items with a delay duration. Items cannot be
// dequeued until their delay has expired.
//
// The implementation uses a priority queue (min-heap) backed by a slice, efficiently
// managing item order based on readiness time.
//
// Blocking operations (Dequeue, DequeueContext) use Go channels and time.Timer to
// wait efficiently without busy-waiting or polling, supporting cancellation via
// context.Context.
//
// Example:
//
//	q := delay.New[string]()
//	q.Enqueue("task", 5 * time.Second)
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
//	defer cancel()
//
//	item, err := q.DequeueContext(ctx)
//	if err != nil {
//	    // handle error (timeout or cancelled)
//	}
//	fmt.Println(item)
package delay
