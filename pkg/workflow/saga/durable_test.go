package saga_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga"
	filestore "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga/adapters/file"
	memstore "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurableExecuteAndResume_Memory(t *testing.T) {
	store := memstore.New()
	reg := saga.NewRegistry()

	var step2Calls atomic.Int32
	s := saga.New("order").
		AddStep(saga.Step{
			Name: "reserve",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return map[string]any{"reserved": true, "qty": data.(map[string]any)["qty"]}, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				return nil, nil
			},
		}).
		AddStep(saga.Step{
			Name: "charge",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				step2Calls.Add(1)
				n := step2Calls.Load()
				if n == 1 {
					// Simulate crash: persist happened for step1; fail before completing step2.
					return nil, errors.New("crash before charge")
				}
				return map[string]any{"charged": true}, nil
			},
			Compensate: func(ctx context.Context, data interface{}) (interface{}, error) {
				return nil, nil
			},
		})
	reg.Register(s)

	exec := saga.NewDurableExecutor(reg, store)

	// First run: step1 completes, step2 fails → compensated
	result, err := exec.Execute(context.Background(), "order", map[string]any{"qty": 2})
	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, saga.StatusCompensated, result.Status)

	// Fresh saga that crashes mid-flight without failing the step action:
	// manually save incomplete state after step1, then Resume.
	store2 := memstore.New()
	reg2 := saga.NewRegistry()
	var chargeRuns atomic.Int32
	s2 := saga.New("pay").
		AddStep(saga.Step{
			Name: "auth",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return map[string]any{"auth": "ok"}, nil
			},
		}).
		AddStep(saga.Step{
			Name: "capture",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				chargeRuns.Add(1)
				return map[string]any{"captured": true}, nil
			},
		})
	reg2.Register(s2)
	ex2 := saga.NewDurableExecutor(reg2, store2)

	okRun, err := ex2.Execute(context.Background(), "pay", map[string]any{"amt": 10})
	require.NoError(t, err)
	assert.Equal(t, saga.StatusCompleted, okRun.Status)
	assert.Equal(t, int32(1), chargeRuns.Load())

	// Crash simulation: save running state after first step only.
	incomplete := &saga.PersistedState{
		ID:            "exec-crash-1",
		SagaName:      "pay",
		Status:        saga.StatusRunning,
		NextStepIndex: 1,
		Input:         map[string]any{"amt": 5},
		CurrentData:   map[string]any{"auth": "ok"},
		Steps: []saga.PersistedStepResult{{
			Name:   "auth",
			Status: saga.StatusCompleted,
			Output: map[string]any{"auth": "ok"},
		}},
	}
	require.NoError(t, store2.Save(context.Background(), incomplete))

	resumed, err := ex2.Resume(context.Background(), "exec-crash-1")
	require.NoError(t, err)
	assert.Equal(t, saga.StatusCompleted, resumed.Status)
	assert.Equal(t, int32(2), chargeRuns.Load())

	loaded, err := store2.Load(context.Background(), "exec-crash-1")
	require.NoError(t, err)
	assert.True(t, loaded.IsTerminal())
}

func TestDurableFileStore_ResumeAfterRestart(t *testing.T) {
	dir := t.TempDir()
	store, err := filestore.New(filepath.Join(dir, "sagas"))
	require.NoError(t, err)

	reg := saga.NewRegistry()
	var calls atomic.Int32
	reg.Register(saga.New("xfer").
		AddStep(saga.Step{
			Name: "debit",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return map[string]any{"debited": true}, nil
			},
		}).
		AddStep(saga.Step{
			Name: "credit",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				calls.Add(1)
				return map[string]any{"credited": true}, nil
			},
		}))

	require.NoError(t, store.Save(context.Background(), &saga.PersistedState{
		ID:            "file-exec-1",
		SagaName:      "xfer",
		Status:        saga.StatusRunning,
		NextStepIndex: 1,
		Input:         map[string]any{"n": 1},
		CurrentData:   map[string]any{"debited": true},
		Steps: []saga.PersistedStepResult{{
			Name:   "debit",
			Status: saga.StatusCompleted,
			Output: map[string]any{"debited": true},
		}},
	}))

	// Simulate process restart with a new executor + same directory.
	store2, err := filestore.New(filepath.Join(dir, "sagas"))
	require.NoError(t, err)
	ex2 := saga.NewDurableExecutor(reg, store2)

	resumed, err := ex2.Resume(context.Background(), "file-exec-1")
	require.NoError(t, err)
	assert.Equal(t, saga.StatusCompleted, resumed.Status)
	assert.Equal(t, int32(1), calls.Load())

	// File still exists and is terminal.
	_, err = os.Stat(filepath.Join(dir, "sagas", "file-exec-1.json"))
	require.NoError(t, err)
}

func TestDurableResumeAll(t *testing.T) {
	store := memstore.New()
	reg := saga.NewRegistry()
	reg.Register(saga.New("one").AddStep(saga.Step{
		Name: "a",
		Action: func(ctx context.Context, data interface{}) (interface{}, error) {
			return "done", nil
		},
	}))
	require.NoError(t, store.Save(context.Background(), &saga.PersistedState{
		ID:            "inc-1",
		SagaName:      "one",
		Status:        saga.StatusRunning,
		NextStepIndex: 0,
		Input:         "x",
		CurrentData:   "x",
	}))

	ex := saga.NewDurableExecutor(reg, store)
	out, err := ex.ResumeAll(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, saga.StatusCompleted, out[0].Status)
}

func TestFileStore_CRUD(t *testing.T) {
	store, err := filestore.New(t.TempDir())
	require.NoError(t, err)

	st := &saga.PersistedState{
		ID:       "abc",
		SagaName: "s",
		Status:   saga.StatusRunning,
		Input:    map[string]any{"k": "v"},
	}
	require.NoError(t, store.Save(context.Background(), st))

	got, err := store.Load(context.Background(), "abc")
	require.NoError(t, err)
	assert.Equal(t, "s", got.SagaName)

	list, err := store.ListIncomplete(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, store.Delete(context.Background(), "abc"))
	_, err = store.Load(context.Background(), "abc")
	require.Error(t, err)
}
