package scheduler

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency/distlock"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// JobStatus represents the status of a job execution.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusSkipped   JobStatus = "skipped"
)

// JobFunc is the function signature for jobs.
type JobFunc func(ctx context.Context) error

// Job represents a scheduled job.
type Job struct {
	// ID is the unique job identifier.
	ID string

	// Name is the job name.
	Name string

	// Schedule is the cron expression or "once" for one-time.
	Schedule string

	// NextRun is the next scheduled run time.
	NextRun time.Time

	// LastRun is the last run time.
	LastRun time.Time

	// LastStatus is the last execution status.
	LastStatus JobStatus

	// Timeout is the job timeout.
	Timeout time.Duration

	// Enabled indicates if the job is active.
	Enabled bool

	// CreatedAt is when the job was created.
	CreatedAt time.Time
}

// JobExecution represents a job execution instance.
type JobExecution struct {
	// ID is the execution ID.
	ID string

	// JobID is the job being executed.
	JobID string

	// Status is the execution status.
	Status JobStatus

	// Error is the error message (if failed).
	Error string

	// StartedAt is when execution started.
	StartedAt time.Time

	// CompletedAt is when execution completed.
	CompletedAt time.Time
}

const defaultLockTTL = 30 * time.Second

// Scheduler manages scheduled jobs.
type Scheduler struct {
	mu         *concurrency.SmartRWMutex
	store      Store
	locker     distlock.Locker
	jobs       map[string]*Job
	handlers   map[string]JobFunc
	executions map[string][]*JobExecution
	running    bool
	stopCh     chan struct{}
	interval   time.Duration
}

// New creates a scheduler.
//
// store may be nil (ephemeral in-memory maps only). Prefer NewMemoryStore for persistence
// across restarts within a process, or a durable Store implementation.
// locker may be nil (no distributed locking; fine for single-node). Pass a
// pkg/concurrency/distlock.Locker (e.g. memory or Redis adapter) so only one
// node runs each due job.
func New(store Store, locker distlock.Locker) *Scheduler {
	return &Scheduler{
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "workflow-scheduler"}),
		store:      store,
		locker:     locker,
		jobs:       make(map[string]*Job),
		handlers:   make(map[string]JobFunc),
		executions: make(map[string][]*JobExecution),
		interval:   time.Minute,
	}
}

// SetTickInterval overrides the polling interval (useful in tests).
func (s *Scheduler) SetTickInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = d
}

// Schedule registers a job with a cron schedule.
func (s *Scheduler) Schedule(name, schedule string, handler JobFunc) error {
	if name == "" {
		return errors.InvalidArgument("job name required", nil)
	}
	if handler == nil {
		return errors.InvalidArgument("job handler required", nil)
	}

	nextRun, err := nextRunTime(schedule, time.Now())
	if err != nil {
		return errors.InvalidArgument("invalid schedule", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job := &Job{
		ID:        uuid.NewString(),
		Name:      name,
		Schedule:  schedule,
		NextRun:   nextRun,
		Timeout:   time.Hour,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	s.jobs[name] = job
	s.handlers[name] = handler

	if s.store != nil {
		if err := s.store.SaveJob(context.Background(), job); err != nil {
			return err
		}
	}

	return nil
}

// ScheduleOnce schedules a one-time job.
func (s *Scheduler) ScheduleOnce(name string, runAt time.Time, handler JobFunc) error {
	if name == "" {
		return errors.InvalidArgument("job name required", nil)
	}
	if handler == nil {
		return errors.InvalidArgument("job handler required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job := &Job{
		ID:        uuid.NewString(),
		Name:      name,
		Schedule:  "once",
		NextRun:   runAt,
		Timeout:   time.Hour,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	s.jobs[name] = job
	s.handlers[name] = handler

	if s.store != nil {
		if err := s.store.SaveJob(context.Background(), job); err != nil {
			return err
		}
	}

	return nil
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.Conflict("scheduler already running", nil)
	}
	s.running = true
	s.stopCh = make(chan struct{})
	interval := s.interval
	s.mu.Unlock()

	go s.run(ctx, interval)
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

func (s *Scheduler) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	s.mu.RLock()
	now := time.Now()
	var dueJobs []*Job

	for _, job := range s.jobs {
		if job.Enabled && !job.NextRun.IsZero() && !job.NextRun.After(now) {
			cp := *job
			dueJobs = append(dueJobs, &cp)
		}
	}
	s.mu.RUnlock()

	for _, job := range dueJobs {
		go s.executeJob(ctx, job.Name)
	}
}

func (s *Scheduler) executeJob(ctx context.Context, name string) {
	s.mu.RLock()
	job, ok := s.jobs[name]
	handler, hasHandler := s.handlers[name]
	s.mu.RUnlock()
	if !ok || !hasHandler {
		return
	}

	// Distributed lock so only one node runs the job.
	if s.locker != nil {
		ttl := job.Timeout
		if ttl <= 0 {
			ttl = defaultLockTTL
		}
		lock := s.locker.NewLock("workflow:scheduler:"+name, ttl)
		acquired, err := lock.Acquire(ctx)
		if err != nil || !acquired {
			return
		}
		defer func() { _ = lock.Release(ctx) }()
	}

	exec := &JobExecution{
		ID:        uuid.NewString(),
		JobID:     job.ID,
		Status:    JobStatusRunning,
		StartedAt: time.Now(),
	}

	execCtx := ctx
	var cancel context.CancelFunc
	if job.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, job.Timeout)
		defer cancel()
	}

	err := handler(execCtx)

	s.mu.Lock()
	job = s.jobs[name]
	if job == nil {
		s.mu.Unlock()
		return
	}

	exec.CompletedAt = time.Now()
	if err != nil {
		exec.Status = JobStatusFailed
		exec.Error = err.Error()
	} else {
		exec.Status = JobStatusCompleted
	}

	job.LastRun = exec.StartedAt
	job.LastStatus = exec.Status

	if job.Schedule != "once" {
		if next, nerr := nextRunTime(job.Schedule, time.Now()); nerr == nil {
			job.NextRun = next
		}
	} else {
		job.Enabled = false
	}

	s.executions[name] = append(s.executions[name], exec)
	jobCopy := *job
	s.mu.Unlock()

	if s.store != nil {
		_ = s.store.SaveJob(ctx, &jobCopy)
		_ = s.store.SaveExecution(ctx, name, exec)
	}
}

// GetJob retrieves a job by name.
func (s *Scheduler) GetJob(name string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[name]
	if !ok {
		return nil, errors.NotFound("job not found", nil)
	}
	cp := *job
	return &cp, nil
}

// ListJobs returns all registered jobs.
func (s *Scheduler) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		cp := *job
		jobs = append(jobs, &cp)
	}
	return jobs
}

// EnableJob enables a job.
func (s *Scheduler) EnableJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[name]
	if !ok {
		return errors.NotFound("job not found", nil)
	}
	job.Enabled = true
	if s.store != nil {
		_ = s.store.SaveJob(context.Background(), job)
	}
	return nil
}

// DisableJob disables a job.
func (s *Scheduler) DisableJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[name]
	if !ok {
		return errors.NotFound("job not found", nil)
	}
	job.Enabled = false
	if s.store != nil {
		_ = s.store.SaveJob(context.Background(), job)
	}
	return nil
}

// RunNow immediately executes a job (still respects distributed locking when configured).
func (s *Scheduler) RunNow(ctx context.Context, name string) (*JobExecution, error) {
	s.mu.RLock()
	job, ok := s.jobs[name]
	handler, hasHandler := s.handlers[name]
	s.mu.RUnlock()

	if !ok || !hasHandler {
		return nil, errors.NotFound("job not found", nil)
	}

	if s.locker != nil {
		ttl := job.Timeout
		if ttl <= 0 {
			ttl = defaultLockTTL
		}
		lock := s.locker.NewLock("workflow:scheduler:"+name, ttl)
		acquired, err := lock.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		if !acquired {
			return nil, errors.Conflict("job already running on another node", nil)
		}
		defer func() { _ = lock.Release(ctx) }()
	}

	exec := &JobExecution{
		ID:        uuid.NewString(),
		JobID:     job.ID,
		Status:    JobStatusRunning,
		StartedAt: time.Now(),
	}

	err := handler(ctx)
	exec.CompletedAt = time.Now()

	s.mu.Lock()
	job = s.jobs[name]
	if err != nil {
		exec.Status = JobStatusFailed
		exec.Error = err.Error()
	} else {
		exec.Status = JobStatusCompleted
	}
	if job != nil {
		job.LastRun = exec.StartedAt
		job.LastStatus = exec.Status
	}
	s.executions[name] = append(s.executions[name], exec)
	s.mu.Unlock()

	if s.store != nil {
		_ = s.store.SaveExecution(ctx, name, exec)
		if job != nil {
			_ = s.store.SaveJob(ctx, job)
		}
	}

	return exec, err
}
