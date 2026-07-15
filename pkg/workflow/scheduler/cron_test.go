package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCron_StandardAndDescriptors(t *testing.T) {
	s := scheduler.New(nil, nil)
	require.NoError(t, s.Schedule("hourly", "@hourly", func(ctx context.Context) error { return nil }))
	require.NoError(t, s.Schedule("every", "@every 1h", func(ctx context.Context) error { return nil }))
	require.NoError(t, s.Schedule("five", "0 0 * * *", func(ctx context.Context) error { return nil }))

	job, err := s.GetJob("hourly")
	require.NoError(t, err)
	assert.False(t, job.NextRun.IsZero())
	assert.True(t, job.NextRun.After(time.Now().Add(-time.Minute)))
}

func TestInstrumentedScheduler_RunNow(t *testing.T) {
	inner := scheduler.New(nil, nil)
	ran := false
	require.NoError(t, inner.ScheduleOnce("once", time.Now().Add(time.Hour), func(ctx context.Context) error {
		ran = true
		return nil
	}))

	inst := scheduler.NewInstrumentedScheduler(inner)
	exec, err := inst.RunNow(context.Background(), "once")
	require.NoError(t, err)
	assert.True(t, ran)
	assert.Equal(t, scheduler.JobStatusCompleted, exec.Status)
	assert.NotNil(t, inst.Underlying())
}

func TestInstrumentedScheduler_InvalidCron(t *testing.T) {
	inst := scheduler.NewInstrumentedScheduler(scheduler.New(nil, nil))
	err := inst.Schedule("bad", "not a cron", func(ctx context.Context) error { return nil })
	require.Error(t, err)
}
