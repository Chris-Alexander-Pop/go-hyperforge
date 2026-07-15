package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm"
	vmmem "github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

func TestEventedVMManager_Lifecycle(t *testing.T) {
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var types []string
	done := make(chan struct{}, 4)
	_, err := bus.Subscribe(context.Background(), vm.TopicVM, func(ctx context.Context, ev events.Event) error {
		types = append(types, ev.Type)
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	mgr := vm.NewEventedVMManager(vmmem.New(), bus)
	ctx := context.Background()
	inst, err := mgr.Create(ctx, vm.CreateOptions{Name: "web", ImageID: "ami-1"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout create")
	}

	if err := mgr.Stop(ctx, inst.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout stop")
	}

	if err := mgr.Start(ctx, inst.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout start")
	}

	if err := mgr.Terminate(ctx, inst.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout terminate")
	}

	want := []string{vm.EventTypeCreated, vm.EventTypeStopped, vm.EventTypeStarted, vm.EventTypeTerminated}
	if len(types) != len(want) {
		t.Fatalf("types=%v", types)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("types=%v want=%v", types, want)
		}
	}
}

func TestEventedVMManager_NilBus(t *testing.T) {
	mgr := vm.NewEventedVMManager(vmmem.New(), nil)
	if _, err := mgr.Create(context.Background(), vm.CreateOptions{Name: "n", ImageID: "i"}); err != nil {
		t.Fatal(err)
	}
}
