package scheduler

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Store persists scheduled jobs and their executions.
// Implementations may back Redis, SQL, or in-process memory.
type Store interface {
	// SaveJob upserts a job by Name.
	SaveJob(ctx context.Context, job *Job) error
	// GetJob retrieves a job by name.
	GetJob(ctx context.Context, name string) (*Job, error)
	// ListJobs returns all persisted jobs.
	ListJobs(ctx context.Context) ([]*Job, error)
	// DeleteJob removes a job by name.
	DeleteJob(ctx context.Context, name string) error
	// SaveExecution appends an execution record for a job name.
	SaveExecution(ctx context.Context, jobName string, exec *JobExecution) error
	// ListExecutions returns executions for a job name.
	ListExecutions(ctx context.Context, jobName string) ([]*JobExecution, error)
}

// MemoryStore is an in-process Store suitable for tests and single-node use.
type MemoryStore struct {
	mu         *concurrency.SmartRWMutex
	jobs       map[string]*Job
	executions map[string][]*JobExecution
}

// NewMemoryStore creates an empty in-memory job store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "workflow-scheduler-store"}),
		jobs:       make(map[string]*Job),
		executions: make(map[string][]*JobExecution),
	}
}

// SaveJob implements Store.
func (s *MemoryStore) SaveJob(ctx context.Context, job *Job) error {
	if job == nil || job.Name == "" {
		return errors.InvalidArgument("job name required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *job
	s.jobs[job.Name] = &cp
	return nil
}

// GetJob implements Store.
func (s *MemoryStore) GetJob(ctx context.Context, name string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[name]
	if !ok {
		return nil, errors.NotFound("job not found", nil)
	}
	cp := *job
	return &cp, nil
}

// ListJobs implements Store.
func (s *MemoryStore) ListJobs(ctx context.Context) ([]*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		cp := *job
		out = append(out, &cp)
	}
	return out, nil
}

// DeleteJob implements Store.
func (s *MemoryStore) DeleteJob(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[name]; !ok {
		return errors.NotFound("job not found", nil)
	}
	delete(s.jobs, name)
	return nil
}

// SaveExecution implements Store.
func (s *MemoryStore) SaveExecution(ctx context.Context, jobName string, exec *JobExecution) error {
	if exec == nil {
		return errors.InvalidArgument("execution required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *exec
	s.executions[jobName] = append(s.executions[jobName], &cp)
	return nil
}

// ListExecutions implements Store.
func (s *MemoryStore) ListExecutions(ctx context.Context, jobName string) ([]*JobExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.executions[jobName]
	out := make([]*JobExecution, 0, len(src))
	for _, exec := range src {
		cp := *exec
		out = append(out, &cp)
	}
	return out, nil
}

var _ Store = (*MemoryStore)(nil)
