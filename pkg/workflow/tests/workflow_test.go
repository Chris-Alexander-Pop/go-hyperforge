package tests

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency/distlock/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	workflowmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/scheduler"
	"github.com/stretchr/testify/suite"
)

// WorkflowEngineSuite tests WorkflowEngine implementations.
type WorkflowEngineSuite struct {
	suite.Suite
	engine workflow.WorkflowEngine
	ctx    context.Context
}

func (s *WorkflowEngineSuite) SetupTest() {
	s.engine = workflowmemory.New()
	s.ctx = context.Background()
}

func (s *WorkflowEngineSuite) TestRegisterAndGetWorkflow() {
	def := workflow.WorkflowDefinition{
		ID:   "order-workflow",
		Name: "Order Processing",
		States: []workflow.State{
			{Name: "validate", Type: "Task", Next: "process"},
			{Name: "process", Type: "Task", End: true},
		},
		StartAt: "validate",
	}

	err := s.engine.RegisterWorkflow(s.ctx, def)
	s.Require().NoError(err)

	got, err := s.engine.GetWorkflow(s.ctx, "order-workflow")
	s.Require().NoError(err)
	s.Equal("Order Processing", got.Name)
}

func (s *WorkflowEngineSuite) TestGetWorkflowNotFound() {
	_, err := s.engine.GetWorkflow(s.ctx, "nonexistent")
	s.ErrorIs(err, workflow.ErrWorkflowNotFound)
}

func (s *WorkflowEngineSuite) TestStartExecution() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "test-wf"})
	s.Require().NoError(err)

	exec, err := s.engine.Start(s.ctx, workflow.StartOptions{
		WorkflowID: "test-wf",
		Input:      map[string]string{"order": "123"},
	})
	s.Require().NoError(err)
	s.NotEmpty(exec.ID)
	s.Equal("test-wf", exec.WorkflowID)
}

func (s *WorkflowEngineSuite) TestStartExecutionWorkflowNotFound() {
	_, err := s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "nonexistent"})
	s.ErrorIs(err, workflow.ErrWorkflowNotFound)
}

func (s *WorkflowEngineSuite) TestGetExecution() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "test-wf"})
	s.Require().NoError(err)
	exec, err := s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "test-wf"})
	s.Require().NoError(err)

	got, err := s.engine.GetExecution(s.ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(exec.ID, got.ID)
}

func (s *WorkflowEngineSuite) TestListExecutions() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "test-wf"})
	s.Require().NoError(err)

	for i := 0; i < 3; i++ {
		_, err := s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "test-wf"})
		s.Require().NoError(err)
	}

	result, err := s.engine.ListExecutions(s.ctx, workflow.ListOptions{})
	s.Require().NoError(err)
	s.Len(result.Executions, 3)
}

func (s *WorkflowEngineSuite) TestCancelExecution() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "test-wf"})
	s.Require().NoError(err)
	exec, err := s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "test-wf"})
	s.Require().NoError(err)

	err = s.engine.Cancel(s.ctx, exec.ID)
	s.Require().NoError(err)

	exec, _ = s.engine.GetExecution(s.ctx, exec.ID)
	s.Equal(workflow.StatusCancelled, exec.Status)
}

func (s *WorkflowEngineSuite) TestWaitForCompletion() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "test-wf"})
	s.Require().NoError(err)
	exec, err := s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "test-wf"})
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()

	result, err := s.engine.Wait(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(workflow.StatusCompleted, result.Status)
}

func (s *WorkflowEngineSuite) TestStartOptionsTimeout() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "timeout-wf"})
	s.Require().NoError(err)

	exec, err := s.engine.Start(s.ctx, workflow.StartOptions{
		WorkflowID: "timeout-wf",
		Timeout:    10 * time.Millisecond, // shorter than simulated work (100ms)
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()

	result, err := s.engine.Wait(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(workflow.StatusTimedOut, result.Status)
}

func (s *WorkflowEngineSuite) TestDuplicateExecutionID() {
	err := s.engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "dup-wf"})
	s.Require().NoError(err)

	_, err = s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "dup-wf", ExecutionID: "same"})
	s.Require().NoError(err)
	_, err = s.engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "dup-wf", ExecutionID: "same"})
	s.ErrorIs(err, workflow.ErrExecutionAlreadyExists)
}

func (s *WorkflowEngineSuite) TestEventedEngineLifecycle() {
	bus := eventsmemory.New(events.Config{})
	defer bus.Close()

	var mu sync.Mutex
	var types []string
	_, err := bus.Subscribe(s.ctx, workflow.TopicWorkflow, func(ctx context.Context, ev events.Event) error {
		mu.Lock()
		types = append(types, ev.Type)
		mu.Unlock()
		return nil
	})
	s.Require().NoError(err)

	engine := workflow.NewEventedEngine(workflowmemory.New(), bus)
	err = engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "evt-wf"})
	s.Require().NoError(err)

	exec, err := engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "evt-wf"})
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()
	_, err = engine.Wait(ctx, exec.ID)
	s.Require().NoError(err)

	mu.Lock()
	defer mu.Unlock()
	s.Contains(types, workflow.EventTypeStarted)
	s.Contains(types, workflow.EventTypeCompleted)
}

func (s *WorkflowEngineSuite) TestEventedEngineNilBus() {
	engine := workflow.NewEventedEngine(workflowmemory.New(), nil)
	err := engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "nil-bus"})
	s.Require().NoError(err)
	exec, err := engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "nil-bus"})
	s.Require().NoError(err)
	s.NotEmpty(exec.ID)
}

func (s *WorkflowEngineSuite) TestInstrumentedEngine() {
	engine := workflow.NewInstrumentedWorkflowEngine(workflowmemory.New())
	err := engine.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{ID: "inst-wf"})
	s.Require().NoError(err)
	exec, err := engine.Start(s.ctx, workflow.StartOptions{WorkflowID: "inst-wf"})
	s.Require().NoError(err)
	s.NotEmpty(exec.ID)
}

func (s *WorkflowEngineSuite) TestStateMachineTaskAndWait() {
	raw := workflowmemory.New()
	eng := raw.(*workflowmemory.Engine)
	eng.RegisterTaskHandler("double", func(ctx context.Context, input interface{}) (interface{}, error) {
		m := input.(map[string]int)
		return map[string]int{"n": m["n"] * 2}, nil
	})

	err := eng.RegisterWorkflow(s.ctx, workflow.WorkflowDefinition{
		ID:      "sm-wf",
		StartAt: "wait-a-bit",
		States: []workflow.State{
			{Name: "wait-a-bit", Type: "Wait", Seconds: 0, Next: "double"},
			{Name: "double", Type: "Task", Resource: "double", End: true},
		},
	})
	s.Require().NoError(err)

	exec, err := eng.Start(s.ctx, workflow.StartOptions{
		WorkflowID:     "sm-wf",
		Input:          map[string]int{"n": 21},
		IdempotencyKey: "sm-once",
	})
	s.Require().NoError(err)

	// Idempotent re-start returns the same execution.
	again, err := eng.Start(s.ctx, workflow.StartOptions{
		WorkflowID:     "sm-wf",
		Input:          map[string]int{"n": 99},
		IdempotencyKey: "sm-once",
	})
	s.Require().NoError(err)
	s.Equal(exec.ID, again.ID)

	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()
	result, err := eng.Wait(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(workflow.StatusCompleted, result.Status)
	out := result.Output.(map[string]int)
	s.Equal(42, out["n"])
}

// SagaSuite tests the Saga pattern.
type SagaSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *SagaSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *SagaSuite) TestSagaSuccess() {
	var steps []string

	orderSaga := saga.New("order-saga").
		AddStep(saga.Step{
			Name: "reserve-inventory",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "reserve")
				return data, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "release")
				return nil, nil
			},
		}).
		AddStep(saga.Step{
			Name: "charge-payment",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "charge")
				return data, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "refund")
				return nil, nil
			},
		})

	exec, err := orderSaga.Execute(s.ctx, "order-123")
	s.Require().NoError(err)
	s.Equal(saga.StatusCompleted, exec.Status)
	s.Equal([]string{"reserve", "charge"}, steps)
}

func (s *SagaSuite) TestSagaCompensation() {
	var steps []string

	orderSaga := saga.New("order-saga").
		AddStep(saga.Step{
			Name: "reserve-inventory",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "reserve")
				return data, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "release")
				return nil, nil
			},
		}).
		AddStep(saga.Step{
			Name: "charge-payment",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "charge")
				return nil, errors.New("payment failed")
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				steps = append(steps, "refund")
				return nil, nil
			},
		})

	exec, err := orderSaga.Execute(s.ctx, "order-123")
	s.Error(err)
	s.Equal(saga.StatusCompensated, exec.Status)
	s.Equal([]string{"reserve", "charge", "release"}, steps)
}

func (s *SagaSuite) TestSagaAggregatesCompensationErrors() {
	orderSaga := saga.New("multi-comp-fail").
		AddStep(saga.Step{
			Name: "step-a",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return data, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				return nil, errors.New("comp-a failed")
			},
		}).
		AddStep(saga.Step{
			Name: "step-b",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return data, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				return nil, errors.New("comp-b failed")
			},
		}).
		AddStep(saga.Step{
			Name: "step-c",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return nil, errors.New("forward failed")
			},
		})

	exec, err := orderSaga.Execute(s.ctx, "x")
	s.Error(err)
	s.Equal(saga.StatusFailed, exec.Status)
	s.Contains(exec.Error, "comp-a failed")
	s.Contains(exec.Error, "comp-b failed")
}

func (s *SagaSuite) TestInstrumentedSaga() {
	inner := saga.New("inst-saga").AddStep(saga.Step{
		Name: "ok",
		Action: func(ctx context.Context, data interface{}) (interface{}, error) {
			return data, nil
		},
	})
	wrapped := saga.NewInstrumentedSaga(inner)
	exec, err := wrapped.Execute(s.ctx, "in")
	s.Require().NoError(err)
	s.Equal(saga.StatusCompleted, exec.Status)
}

func (s *SagaSuite) TestSagaRegistry() {
	reg := saga.NewRegistry()
	sg := saga.New("reg-saga")
	reg.Register(sg)
	got, ok := reg.Get("reg-saga")
	s.True(ok)
	s.Equal(sg, got)
	_, ok = reg.Get("missing")
	s.False(ok)
}

// SchedulerSuite tests the job scheduler.
type SchedulerSuite struct {
	suite.Suite
	sched *scheduler.Scheduler
	ctx   context.Context
}

func (s *SchedulerSuite) SetupTest() {
	s.sched = scheduler.New(scheduler.NewMemoryStore(), memory.New())
	s.ctx = context.Background()
}

func (s *SchedulerSuite) TestScheduleJob() {
	err := s.sched.Schedule("test-job", "@hourly", func(ctx context.Context) error {
		return nil
	})
	s.Require().NoError(err)

	job, err := s.sched.GetJob("test-job")
	s.Require().NoError(err)
	s.Equal("test-job", job.Name)
	s.True(job.Enabled)
}

func (s *SchedulerSuite) TestScheduleCronExpression() {
	err := s.sched.Schedule("cron-job", "0 0 * * *", func(ctx context.Context) error {
		return nil
	})
	s.Require().NoError(err)

	job, err := s.sched.GetJob("cron-job")
	s.Require().NoError(err)
	s.Equal("0 0 * * *", job.Schedule)
	s.False(job.NextRun.IsZero())
	s.True(job.NextRun.After(time.Now().Add(-time.Second)))
}

func (s *SchedulerSuite) TestScheduleEvery() {
	err := s.sched.Schedule("every-job", "@every 1h", func(ctx context.Context) error {
		return nil
	})
	s.Require().NoError(err)
	job, err := s.sched.GetJob("every-job")
	s.Require().NoError(err)
	s.WithinDuration(time.Now().Add(time.Hour), job.NextRun, 2*time.Second)
}

func (s *SchedulerSuite) TestScheduleInvalidCron() {
	err := s.sched.Schedule("bad", "not a cron", func(ctx context.Context) error { return nil })
	s.Error(err)
}

func (s *SchedulerSuite) TestScheduleOnce() {
	runAt := time.Now().Add(time.Hour)
	err := s.sched.ScheduleOnce("once-job", runAt, func(ctx context.Context) error {
		return nil
	})
	s.Require().NoError(err)

	job, err := s.sched.GetJob("once-job")
	s.Require().NoError(err)
	s.Equal("once", job.Schedule)
}

func (s *SchedulerSuite) TestListJobs() {
	for i := 0; i < 3; i++ {
		err := s.sched.Schedule("job-"+string(rune('a'+i)), "@daily", func(ctx context.Context) error { return nil })
		s.Require().NoError(err)
	}

	jobs := s.sched.ListJobs()
	s.Len(jobs, 3)
}

func (s *SchedulerSuite) TestEnableDisableJob() {
	err := s.sched.Schedule("toggle-job", "@hourly", func(ctx context.Context) error { return nil })
	s.Require().NoError(err)

	err = s.sched.DisableJob("toggle-job")
	s.Require().NoError(err)

	job, _ := s.sched.GetJob("toggle-job")
	s.False(job.Enabled)

	err = s.sched.EnableJob("toggle-job")
	s.Require().NoError(err)

	job, _ = s.sched.GetJob("toggle-job")
	s.True(job.Enabled)
}

func (s *SchedulerSuite) TestRunNow() {
	executed := false
	err := s.sched.Schedule("run-now-job", "@daily", func(ctx context.Context) error {
		executed = true
		return nil
	})
	s.Require().NoError(err)

	exec, err := s.sched.RunNow(s.ctx, "run-now-job")
	s.Require().NoError(err)
	s.True(executed)
	s.Equal(scheduler.JobStatusCompleted, exec.Status)
}

func (s *SchedulerSuite) TestRunNowWithError() {
	err := s.sched.Schedule("fail-job", "@daily", func(ctx context.Context) error {
		return errors.New("job failed")
	})
	s.Require().NoError(err)

	exec, err := s.sched.RunNow(s.ctx, "fail-job")
	s.Error(err)
	s.Equal(scheduler.JobStatusFailed, exec.Status)
}

func (s *SchedulerSuite) TestNilStoreAndLocker() {
	sched := scheduler.New(nil, nil)
	err := sched.Schedule("local", "@hourly", func(ctx context.Context) error { return nil })
	s.Require().NoError(err)
	exec, err := sched.RunNow(s.ctx, "local")
	s.Require().NoError(err)
	s.Equal(scheduler.JobStatusCompleted, exec.Status)
}

func (s *SchedulerSuite) TestDistlockSkipsDuplicateRun() {
	locker := memory.New()
	sched := scheduler.New(nil, locker)

	var runs atomic.Int32
	err := sched.Schedule("locked", "@hourly", func(ctx context.Context) error {
		runs.Add(1)
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	s.Require().NoError(err)

	// Hold the same lock key the scheduler uses so RunNow cannot acquire it.
	lock := locker.NewLock("workflow:scheduler:locked", time.Second)
	ok, err := lock.Acquire(s.ctx)
	s.Require().NoError(err)
	s.True(ok)

	_, err = sched.RunNow(s.ctx, "locked")
	s.Error(err)
	s.Equal(int32(0), runs.Load())

	_ = lock.Release(s.ctx)
	_, err = sched.RunNow(s.ctx, "locked")
	s.Require().NoError(err)
	s.Equal(int32(1), runs.Load())
}

func (s *SchedulerSuite) TestMemoryStorePersistence() {
	store := scheduler.NewMemoryStore()
	sched := scheduler.New(store, nil)
	err := sched.Schedule("persisted", "30 4 * * *", func(ctx context.Context) error { return nil })
	s.Require().NoError(err)

	job, err := store.GetJob(s.ctx, "persisted")
	s.Require().NoError(err)
	s.Equal("30 4 * * *", job.Schedule)

	_, err = sched.RunNow(s.ctx, "persisted")
	s.Require().NoError(err)
	execs, err := store.ListExecutions(s.ctx, "persisted")
	s.Require().NoError(err)
	s.Len(execs, 1)
}

func TestWorkflowEngineSuite(t *testing.T) {
	suite.Run(t, new(WorkflowEngineSuite))
}

func TestSagaSuite(t *testing.T) {
	suite.Run(t, new(SagaSuite))
}

func TestSchedulerSuite(t *testing.T) {
	suite.Run(t, new(SchedulerSuite))
}
