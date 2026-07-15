package saga_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga"
	memstore "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstrumentedDurableExecutor(t *testing.T) {
	reg := saga.NewRegistry()
	s := saga.New("pay").AddStep(saga.Step{
		Name: "charge",
		Action: func(ctx context.Context, data interface{}) (interface{}, error) {
			return "ok", nil
		},
	})
	reg.Register(s)

	store := memstore.New()
	exec := saga.NewInstrumentedDurableExecutor(saga.NewDurableExecutor(reg, store))

	out, err := exec.Execute(context.Background(), "pay", map[string]string{"amt": "1"})
	require.NoError(t, err)
	assert.Equal(t, saga.StatusCompleted, out.Status)

	resumed, err := exec.Resume(context.Background(), out.ID)
	require.NoError(t, err)
	assert.Equal(t, saga.StatusCompleted, resumed.Status)

	all, err := exec.ResumeAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, all)
}
