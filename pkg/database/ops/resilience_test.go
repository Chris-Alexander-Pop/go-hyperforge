package ops_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/database/ops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRetry_SucceedsAfterFailures(t *testing.T) {
	var calls atomic.Int32
	err := ops.WithRetry(context.Background(), 3, time.Millisecond, func() error {
		if calls.Add(1) < 3 {
			return errors.New("transient")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestWithRetry_ExhaustsAttempts(t *testing.T) {
	err := ops.WithRetry(context.Background(), 2, time.Millisecond, func() error {
		return errors.New("always fail")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
}

func TestWithRetry_RespectsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ops.WithRetry(ctx, 5, time.Second, func() error {
		return errors.New("fail")
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
