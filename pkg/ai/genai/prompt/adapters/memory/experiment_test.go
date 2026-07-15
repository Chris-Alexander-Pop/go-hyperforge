package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestABExperimentStickyAssign(t *testing.T) {
	ab := memory.NewABExperiment()
	require.NoError(t, ab.RegisterVariants("exp1", []prompt.Variant{
		{ID: "a", TemplateName: "greet", Version: "1", Weight: 1},
		{ID: "b", TemplateName: "greet", Version: "2", Weight: 1},
	}))
	ctx := context.Background()
	v1, err := ab.Assign(ctx, "exp1", "user-1")
	require.NoError(t, err)
	v2, err := ab.Assign(ctx, "exp1", "user-1")
	require.NoError(t, err)
	require.Equal(t, v1.ID, v2.ID)
	require.NoError(t, ab.RecordOutcome(ctx, "exp1", "user-1", 1.0))
}

func TestRemoteRegistryFetchAndSync(t *testing.T) {
	reg := memory.NewRemoteRegistry()
	reg.Put(prompt.Template{Name: "greet", Version: "1", Body: "hi {{name}}"})
	reg.Put(prompt.Template{Name: "greet", Version: "2", Body: "hello {{name}}"})
	ctx := context.Background()

	t2, err := reg.Fetch(ctx, "greet", "2")
	require.NoError(t, err)
	require.Equal(t, "hello {{name}}", t2.Body)

	latest, err := reg.Fetch(ctx, "greet", "latest")
	require.NoError(t, err)
	require.Equal(t, "2", latest.Version)

	list, err := reg.List(ctx, "greet")
	require.NoError(t, err)
	require.Len(t, list, 2)

	store := memory.New()
	require.NoError(t, reg.SyncToStore(ctx, store))
	got, err := store.Get(ctx, "greet", "2")
	require.NoError(t, err)
	require.Equal(t, "hello {{name}}", got.Body)
}
