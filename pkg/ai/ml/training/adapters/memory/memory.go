// Package memory provides an in-memory training.Trainer stub for tests.
package memory

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Ensure Trainer implements training.Trainer.
var _ training.Trainer = (*Trainer)(nil)

// Trainer is an in-memory training backend.
type Trainer struct {
	mu      *concurrency.SmartRWMutex
	jobs    map[string]*training.Job
	metrics map[string][]training.Metrics
	logs    map[string][]string
	cps     map[string][]training.Checkpoint
	seq     atomic.Uint64
	closed  atomic.Bool
}

// New creates an empty in-memory trainer.
func New() *Trainer {
	return &Trainer{
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "ml-training-memory"}),
		jobs:    make(map[string]*training.Job),
		metrics: make(map[string][]training.Metrics),
		logs:    make(map[string][]string),
		cps:     make(map[string][]training.Checkpoint),
	}
}

// StartJob implements training.Trainer.
func (t *Trainer) StartJob(ctx context.Context, config training.JobConfig) (*training.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if t.closed.Load() {
		return nil, training.ErrClosed
	}
	if config.Model == "" && config.EntryPoint == "" {
		return nil, training.ErrInvalidJob
	}
	id := fmt.Sprintf("job-%d", t.seq.Add(1))
	name := config.Name
	if name == "" {
		name = id
	}
	now := time.Now().UTC()
	job := &training.Job{
		ID:         id,
		Name:       name,
		Status:     training.StatusRunning,
		CreatedAt:  now,
		StartedAt:  &now,
		Config:     config,
		OutputPath: config.OutputPath,
	}
	t.mu.Lock()
	t.jobs[id] = job
	t.logs[id] = []string{"started"}
	t.metrics[id] = []training.Metrics{{
		Step: 0, Epoch: 0, Loss: 1.0, Timestamp: now,
	}}
	t.mu.Unlock()
	cp := *job
	return &cp, nil
}

// GetJob implements training.Trainer.
func (t *Trainer) GetJob(ctx context.Context, jobID string) (*training.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	j, ok := t.jobs[jobID]
	if !ok {
		return nil, training.ErrJobNotFound
	}
	cp := *j
	return &cp, nil
}

// ListJobs implements training.Trainer.
func (t *Trainer) ListJobs(ctx context.Context) ([]*training.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]*training.Job, 0, len(t.jobs))
	for _, j := range t.jobs {
		cp := *j
		out = append(out, &cp)
	}
	return out, nil
}

// StopJob implements training.Trainer.
func (t *Trainer) StopJob(ctx context.Context, jobID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	j, ok := t.jobs[jobID]
	if !ok {
		return training.ErrJobNotFound
	}
	if j.Status != training.StatusRunning && j.Status != training.StatusPending {
		return training.ErrNotRunning
	}
	now := time.Now().UTC()
	j.Status = training.StatusStopped
	j.CompletedAt = &now
	return nil
}

// GetMetrics implements training.Trainer.
func (t *Trainer) GetMetrics(ctx context.Context, jobID string) ([]training.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if _, ok := t.jobs[jobID]; !ok {
		return nil, training.ErrJobNotFound
	}
	return append([]training.Metrics(nil), t.metrics[jobID]...), nil
}

// GetLogs implements training.Trainer.
func (t *Trainer) GetLogs(ctx context.Context, jobID string, tail int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	logs, ok := t.logs[jobID]
	if !ok {
		return nil, training.ErrJobNotFound
	}
	if tail <= 0 || tail >= len(logs) {
		return append([]string(nil), logs...), nil
	}
	return append([]string(nil), logs[len(logs)-tail:]...), nil
}

// ListCheckpoints implements training.Trainer.
func (t *Trainer) ListCheckpoints(ctx context.Context, jobID string) ([]training.Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if _, ok := t.jobs[jobID]; !ok {
		return nil, training.ErrJobNotFound
	}
	return append([]training.Checkpoint(nil), t.cps[jobID]...), nil
}

// CompleteJob marks a job completed (test helper).
func (t *Trainer) CompleteJob(jobID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	j, ok := t.jobs[jobID]
	if !ok {
		return training.ErrJobNotFound
	}
	now := time.Now().UTC()
	j.Status = training.StatusCompleted
	j.CompletedAt = &now
	t.cps[jobID] = append(t.cps[jobID], training.Checkpoint{
		ID: jobID + "-ckpt-1", Step: 100, Path: "/tmp/" + jobID, CreatedAt: now,
	})
	return nil
}
