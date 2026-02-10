package memory

import (
	"context"
	"math/rand"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/scheduler"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// MemoryScheduler implements a simple random or first-fit scheduler over a static list of hosts.
type MemoryScheduler struct {
	hostIDs []string
	mu      *concurrency.SmartRWMutex
}

// New creates a new MemoryScheduler.
// New creates a new MemoryScheduler.
func New() scheduler.Scheduler {
	return &MemoryScheduler{
		hostIDs: []string{},
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-scheduler",
		}),
	}
}

// AddHost adds a host to the scheduler's pool (for testing).
func (s *MemoryScheduler) AddHost(hostID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostIDs = append(s.hostIDs, hostID)
}

func (s *MemoryScheduler) SelectHost(ctx context.Context, req scheduler.Requirement) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.hostIDs) == 0 {
		return "", scheduler.ErrNoHostFound
	}

	// Simple random selection for memory adapter
	// In a real system, this would check capacity vs requirement
	idx := rand.Intn(len(s.hostIDs))
	return s.hostIDs[idx], nil
}
