package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
	blockmem "github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block/adapters/memory"
)

func TestEventedVolumeStore_CreateDeleteAttach(t *testing.T) {
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var types []string
	done := make(chan struct{}, 4)
	_, err := bus.Subscribe(context.Background(), block.TopicBlock, func(ctx context.Context, ev events.Event) error {
		types = append(types, ev.Type)
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	store := block.NewEventedVolumeStore(blockmem.New(), bus)
	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{Name: "data", SizeGB: 10})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout create")
	}

	if err := store.AttachVolume(ctx, block.AttachVolumeOptions{
		VolumeID:   vol.ID,
		InstanceID: "i-1",
		Device:     "/dev/sdf",
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout attach")
	}

	if err := store.DetachVolume(ctx, vol.ID, "i-1"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout detach")
	}

	if err := store.DeleteVolume(ctx, vol.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout delete")
	}

	want := []string{
		block.EventTypeVolumeCreated,
		block.EventTypeVolumeAttached,
		block.EventTypeVolumeDetached,
		block.EventTypeVolumeDeleted,
	}
	if len(types) != len(want) {
		t.Fatalf("types=%v", types)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("types=%v want=%v", types, want)
		}
	}
}

func TestEventedVolumeStore_NilBus(t *testing.T) {
	store := block.NewEventedVolumeStore(blockmem.New(), nil)
	if _, err := store.CreateVolume(context.Background(), block.CreateVolumeOptions{Name: "v", SizeGB: 1}); err != nil {
		t.Fatal(err)
	}
}
