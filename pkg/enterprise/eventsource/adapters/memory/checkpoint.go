package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
)

var _ eventsource.CheckpointStore = (*CheckpointStore)(nil)

// CheckpointStore is an in-memory projection checkpoint store.
type CheckpointStore struct {
	data map[string]eventsource.Checkpoint
	mu   *concurrency.SmartRWMutex
}

// NewCheckpointStore creates an empty in-memory checkpoint store.
func NewCheckpointStore() *CheckpointStore {
	return &CheckpointStore{
		data: make(map[string]eventsource.Checkpoint),
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "eventsource-checkpoint-memory"}),
	}
}

// Load returns the checkpoint for name, or a zero-value checkpoint when missing.
func (s *CheckpointStore) Load(ctx context.Context, name string) (eventsource.Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return eventsource.Checkpoint{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp, ok := s.data[name]
	if !ok {
		return eventsource.Checkpoint{Name: name}, nil
	}
	return cp, nil
}

// Save upserts the checkpoint.
func (s *CheckpointStore) Save(ctx context.Context, cp eventsource.Checkpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cp.Name == "" {
		return eventsource.ErrInvalidArgument("checkpoint name is required", nil)
	}
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[cp.Name] = cp
	return nil
}
