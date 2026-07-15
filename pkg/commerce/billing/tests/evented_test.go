package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/billing"
	billingmem "github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/billing/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

func TestEventedService_SubscriptionAndInvoice(t *testing.T) {
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var types []string
	done := make(chan struct{}, 2)
	_, err := bus.Subscribe(context.Background(), billing.TopicBilling, func(ctx context.Context, ev events.Event) error {
		types = append(types, ev.Type)
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := billing.NewEventedService(billingmem.New(), bus)
	sub, err := svc.CreateSubscription(context.Background(), "cust1", "basic_monthly")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout create")
	}

	_, err = svc.CreateInvoice(context.Background(), "cust1", commerce.NewMoney(500, "USD"))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout invoice")
	}

	if _, err := svc.CancelSubscription(context.Background(), sub.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout cancel")
	}

	want := map[string]bool{
		billing.EventTypeSubscriptionCreated: true,
		billing.EventTypeInvoiceCreated:      true,
		billing.EventTypeSubscriptionCanceled: true,
	}
	for _, typ := range types {
		if !want[typ] {
			t.Fatalf("unexpected type %s in %v", typ, types)
		}
	}
}

func TestEventedService_NilBus(t *testing.T) {
	svc := billing.NewEventedService(billingmem.New(), nil)
	if _, err := svc.CreateSubscription(context.Background(), "c", "basic_monthly"); err != nil {
		t.Fatal(err)
	}
}
