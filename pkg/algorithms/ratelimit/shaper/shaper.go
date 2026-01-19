package shaper

import (
	"sync"
	"time"
)

// Shaper implements a Leaky Bucket traffic shaper.
// It smooths out bursty traffic by delaying requests.
type Shaper struct {
	rate       float64 // requests per second
	capacity   float64 // bucket size (burst tolerance)
	queue      []func()
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
	stopCh     chan struct{}
}

func New(rate, capacity float64) *Shaper {
	s := &Shaper{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
		stopCh:     make(chan struct{}),
	}
	go s.loop()
	return s
}

// Push adds a task to be executed.
func (s *Shaper) Push(task func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, task)
}

func (s *Shaper) loop() {
	ticker := time.NewTicker(time.Millisecond * 10) // 100hz check
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.process()
		}
	}
}

func (s *Shaper) process() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(s.lastRefill).Seconds()

	// Refill
	s.tokens += elapsed * s.rate
	if s.tokens > s.capacity {
		s.tokens = s.capacity
	}
	s.lastRefill = now

	// Drain queue
	for len(s.queue) > 0 && s.tokens >= 1.0 {
		task := s.queue[0]
		s.queue = s.queue[1:]

		s.tokens -= 1.0
		// Execute task (async or sync? Async usually for shaper)
		go task()
	}
}

func (s *Shaper) Stop() {
	close(s.stopCh)
}
