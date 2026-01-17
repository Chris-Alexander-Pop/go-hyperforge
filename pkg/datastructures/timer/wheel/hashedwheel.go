package wheel

import (
	"container/list"
	"sync"
	"time"
)

// Timer is a Hashed Wheel Timer for efficient O(1) scheduling of timeouts.
type Timer struct {
	tickDuration time.Duration
	wheelSize    int
	wheel        []*list.List
	currentTick  int
	stop         chan struct{}
	mu           sync.Mutex
	wg           sync.WaitGroup
}

type task struct {
	rounds   int
	callback func()
}

// New creates a new Hashed Wheel Timer.
func New(tickDuration time.Duration, wheelSize int) *Timer {
	wheel := make([]*list.List, wheelSize)
	for i := 0; i < wheelSize; i++ {
		wheel[i] = list.New()
	}

	return &Timer{
		tickDuration: tickDuration,
		wheelSize:    wheelSize,
		wheel:        wheel,
		stop:         make(chan struct{}),
	}
}

// Start starts the timer loop.
func (t *Timer) Start() {
	t.wg.Add(1)
	go t.loop()
}

// Stop stops the timer loop.
func (t *Timer) Stop() {
	close(t.stop)
	t.wg.Wait()
}

// Schedule schedules a task to run after the given delay.
func (t *Timer) Schedule(d time.Duration, callback func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Calculate ticks needed
	ticks := int(d / t.tickDuration)
	if ticks < 0 {
		ticks = 0
	}

	rounds := ticks / t.wheelSize
	bucket := (t.currentTick + ticks) % t.wheelSize

	t.wheel[bucket].PushBack(&task{
		rounds:   rounds,
		callback: callback,
	})
}

func (t *Timer) loop() {
	defer t.wg.Done()
	ticker := time.NewTicker(t.tickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.tick()
		}
	}
}

func (t *Timer) tick() {
	t.mu.Lock()
	bucket := t.wheel[t.currentTick]
	var next *list.Element
	for e := bucket.Front(); e != nil; e = next {
		next = e.Next()
		tsk := e.Value.(*task)
		if tsk.rounds > 0 {
			tsk.rounds--
		} else {
			// Fire
			go tsk.callback() // Run async to prevent blocking loop
			bucket.Remove(e)
		}
	}
	t.currentTick = (t.currentTick + 1) % t.wheelSize
	t.mu.Unlock()
}
