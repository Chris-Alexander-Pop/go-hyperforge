package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestMemoryTrainerLifecycle(t *testing.T) {
	tr := training.NewInstrumentedTrainer(memory.New())
	ctx := context.Background()

	job, err := tr.StartJob(ctx, training.JobConfig{Name: "demo", Model: "tiny"})
	require.NoError(t, err)
	require.Equal(t, training.StatusRunning, job.Status)

	got, err := tr.GetJob(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, job.ID, got.ID)

	jobs, err := tr.ListJobs(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	metrics, err := tr.GetMetrics(ctx, job.ID)
	require.NoError(t, err)
	require.NotEmpty(t, metrics)

	logs, err := tr.GetLogs(ctx, job.ID, 10)
	require.NoError(t, err)
	require.Contains(t, logs, "started")

	require.NoError(t, tr.StopJob(ctx, job.ID))
	got, err = tr.GetJob(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, training.StatusStopped, got.Status)

	_, err = tr.GetJob(ctx, "missing")
	require.ErrorIs(t, err, training.ErrJobNotFound)
}
