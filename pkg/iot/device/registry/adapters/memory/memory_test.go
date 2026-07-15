package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/registry"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/registry/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestDeviceRegistryCRUD(t *testing.T) {
	reg := memory.New()
	t.Cleanup(func() { _ = reg.Close() })
	ctx := context.Background()

	d, err := reg.Register(ctx, registry.RegisterOptions{
		ID:         "dev-1",
		Name:       "thermostat",
		ThingType:  "sensor",
		Attributes: map[string]string{"room": "kitchen"},
	})
	require.NoError(t, err)
	require.Equal(t, registry.StatusProvisioned, d.Status)

	got, err := reg.Get(ctx, "dev-1")
	require.NoError(t, err)
	require.Equal(t, "thermostat", got.Name)

	active := registry.StatusActive
	name := "thermo-v2"
	updated, err := reg.Update(ctx, "dev-1", registry.UpdateOptions{Status: &active, Name: &name})
	require.NoError(t, err)
	require.Equal(t, registry.StatusActive, updated.Status)
	require.Equal(t, "thermo-v2", updated.Name)

	require.NoError(t, reg.Touch(ctx, "dev-1"))
	touched, err := reg.Get(ctx, "dev-1")
	require.NoError(t, err)
	require.False(t, touched.LastSeenAt.IsZero())

	list, err := reg.List(ctx, registry.ListOptions{ThingType: "sensor", Status: registry.StatusActive})
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, err = reg.Register(ctx, registry.RegisterOptions{ID: "dev-1", Name: "dup"})
	require.ErrorIs(t, err, registry.ErrDeviceAlreadyExists)

	require.NoError(t, reg.Deregister(ctx, "dev-1"))
	_, err = reg.Get(ctx, "dev-1")
	require.ErrorIs(t, err, registry.ErrDeviceNotFound)
}

func TestDeviceRegistryInvalid(t *testing.T) {
	reg := memory.New()
	_, err := reg.Register(context.Background(), registry.RegisterOptions{})
	require.ErrorIs(t, err, registry.ErrInvalidDevice)
}
