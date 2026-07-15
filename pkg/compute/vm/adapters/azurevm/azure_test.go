package azurevm

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestNewRequiresIDs(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := New(Config{SubscriptionID: "sub"}); err == nil {
		t.Fatal("expected resource group error")
	}
}

func TestUnimplementedOps(t *testing.T) {
	mgr, err := New(Config{SubscriptionID: "sub", ResourceGroup: "rg"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ops := []error{
		func() error { _, e := mgr.Create(ctx, vm.CreateOptions{}); return e }(),
		func() error { _, e := mgr.Get(ctx, "x"); return e }(),
		func() error { _, e := mgr.List(ctx, vm.ListOptions{}); return e }(),
		mgr.Start(ctx, "x"),
		mgr.Stop(ctx, "x"),
		mgr.Reboot(ctx, "x"),
		mgr.Terminate(ctx, "x"),
		mgr.UpdateTags(ctx, "x", nil),
		func() error { _, e := mgr.GetConsoleOutput(ctx, "x"); return e }(),
	}
	for i, err := range ops {
		if !pkgerrors.IsCode(err, pkgerrors.CodeUnimplemented) {
			t.Fatalf("op %d: expected Unimplemented, got %v", i, err)
		}
	}
}
